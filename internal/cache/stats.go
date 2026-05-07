package cache

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// DomainStat holds cache hit/miss counts for a single HA domain.
type DomainStat struct {
	Domain string
	Hits   int64
	Misses int64
	Keys   int // populated externally by EntityMirror.KeyCountByDomain()
}

// Stats tracks cache hit/miss counts per HA domain using lock-free atomic counters.
// Counts are persisted to a JetStream KV bucket every 60 seconds.
type Stats struct {
	kv     jetstream.KeyValue
	counts sync.Map // key: "hits.{domain}" | "misses.{domain}", value: *atomic.Int64
}

// New creates a Stats tracker backed by the given KV bucket.
// Pass nil for kv in tests to skip all KV I/O.
func New(kv jetstream.KeyValue) *Stats {
	return &Stats{kv: kv}
}

// Load reads existing counts from the KV bucket into memory.
// Call once at startup before StartFlusher.
func (s *Stats) Load(ctx context.Context) error {
	if s.kv == nil {
		return nil
	}
	keys, err := s.kv.Keys(ctx)
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			return nil
		}
		return err
	}
	for _, key := range keys {
		entry, err := s.kv.Get(ctx, key)
		if err != nil {
			continue
		}
		n, err := strconv.ParseInt(string(entry.Value()), 10, 64)
		if err != nil || n == 0 {
			continue
		}
		s.counterFor(key).Store(n)
	}
	return nil
}

// Record atomically increments the hit or miss counter for the given domain.
func (s *Stats) Record(domain string, hit bool) {
	s.counterFor(s.counterKey(domain, hit)).Add(1)
}

// Snapshot returns the current counts for all domains, sorted by domain name.
func (s *Stats) Snapshot() []DomainStat {
	byDomain := make(map[string]*DomainStat)
	s.counts.Range(func(k, v any) bool {
		key := k.(string)
		n := v.(*atomic.Int64).Load()
		kind, domain, ok := strings.Cut(key, ".")
		if !ok {
			return true
		}
		if _, exists := byDomain[domain]; !exists {
			byDomain[domain] = &DomainStat{Domain: domain}
		}
		switch kind {
		case "hits":
			byDomain[domain].Hits = n
		case "misses":
			byDomain[domain].Misses = n
		}
		return true
	})
	result := make([]DomainStat, 0, len(byDomain))
	for _, v := range byDomain {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Domain < result[j].Domain
	})
	return result
}

// Reset zeros all in-memory counters and writes zeros to KV.
func (s *Stats) Reset(ctx context.Context) error {
	s.counts.Range(func(k, v any) bool {
		v.(*atomic.Int64).Store(0)
		if s.kv != nil {
			s.kv.PutString(ctx, k.(string), "0")
		}
		return true
	})
	return nil
}

// StartFlusher starts a background goroutine that writes counters to KV every 60s.
func (s *Stats) StartFlusher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.flush(ctx)
			}
		}
	}()
}

func (s *Stats) flush(ctx context.Context) {
	if s.kv == nil {
		return
	}
	s.counts.Range(func(k, v any) bool {
		val := v.(*atomic.Int64).Load()
		s.kv.PutString(ctx, k.(string), strconv.FormatInt(val, 10))
		return true
	})
}

func (s *Stats) counterKey(domain string, hit bool) string {
	if hit {
		return "hits." + domain
	}
	return "misses." + domain
}

func (s *Stats) counterFor(key string) *atomic.Int64 {
	v, _ := s.counts.LoadOrStore(key, &atomic.Int64{})
	return v.(*atomic.Int64)
}
