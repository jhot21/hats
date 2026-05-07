package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// EntityMirror maintains a live in-memory snapshot of HA entity states from
// the JetStream KV cache. A single persistent WatchAll subscription receives
// push updates so HTTP handlers can filter with zero NATS calls per request.
type EntityMirror struct {
	kv      jetstream.KeyValue
	logger  *slog.Logger
	mu      sync.RWMutex
	entries map[string]CacheEntry // keyed by entityID; TTLPct not stored
}

// NewEntityMirror creates a mirror backed by the given KV bucket.
func NewEntityMirror(kv jetstream.KeyValue, logger *slog.Logger) *EntityMirror {
	return &EntityMirror{
		kv:      kv,
		logger:  logger,
		entries: make(map[string]CacheEntry),
	}
}

// Start launches the background watcher loop. Cancel ctx to stop.
func (m *EntityMirror) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *EntityMirror) run(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		snapshotOK, err := m.watch(ctx)
		if snapshotOK {
			backoff = time.Second // reset after a successful connection
		}
		if err != nil {
			m.logger.Error("entity mirror: watcher error, retrying", "error", err, "backoff", backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff = min(backoff*2, 30*time.Second)
		}
	}
}

// watch opens a WatchAll subscription and processes updates until ctx is
// cancelled or the channel closes. Returns (true, nil) on clean ctx exit
// after the snapshot was delivered; (false, err) on connection failure.
func (m *EntityMirror) watch(ctx context.Context) (snapshotOK bool, err error) {
	watcher, err := m.kv.WatchAll(ctx)
	if err != nil {
		return false, fmt.Errorf("WatchAll: %w", err)
	}
	defer watcher.Stop()

	pending := make(map[string]CacheEntry)
	snapshotDone := false

	for {
		select {
		case <-ctx.Done():
			return snapshotDone, nil
		case entry, ok := <-watcher.Updates():
			if !ok {
				return snapshotDone, fmt.Errorf("watcher channel closed")
			}
			if entry == nil {
				// Initial snapshot complete — atomically swap the map.
				// Old entries stay live until this point so handlers
				// never see an empty result mid-reconnect.
				m.mu.Lock()
				m.entries = pending
				m.mu.Unlock()
				snapshotDone = true
				pending = nil // allow GC
				continue
			}
			m.applyEntry(entry, pending, snapshotDone)
		}
	}
}

func (m *EntityMirror) applyEntry(entry jetstream.KeyValueEntry, pending map[string]CacheEntry, live bool) {
	key := entry.Key()
	if !strings.HasPrefix(key, haStatePrefix) {
		return
	}
	entityID := strings.TrimPrefix(key, haStatePrefix)
	domain, _, _ := strings.Cut(entityID, ".")

	if entry.Operation() == jetstream.KeyValuePut {
		ce := CacheEntry{
			Domain:    domain,
			EntityID:  entityID,
			State:     string(entry.Value()),
			UpdatedAt: entry.Created(),
		}
		if !live {
			pending[entityID] = ce
		} else {
			m.mu.Lock()
			m.entries[entityID] = ce
			m.mu.Unlock()
		}
	} else {
		// KeyValueDelete or KeyValuePurge — entry expired or was removed
		if !live {
			delete(pending, entityID)
		} else {
			m.mu.Lock()
			delete(m.entries, entityID)
			m.mu.Unlock()
		}
	}
}

// All returns all entries sorted by UpdatedAt descending.
func (m *EntityMirror) All() []CacheEntry {
	return m.Filtered("", "")
}

// Filtered returns entries matching domain (exact) and q (case-insensitive
// EntityID substring), sorted by UpdatedAt descending. TTLPct is computed
// fresh on each call so it never goes stale in the map.
func (m *EntityMirror) Filtered(domain, q string) []CacheEntry {
	m.mu.RLock()
	snapshot := m.entries
	m.mu.RUnlock()

	now := time.Now()
	q = strings.ToLower(q)
	out := make([]CacheEntry, 0, len(snapshot))
	for _, e := range snapshot {
		if domain != "" && e.Domain != domain {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.EntityID), q) {
			continue
		}
		age := now.Sub(e.UpdatedAt)
		e.TTLPct = max(0, 1.0-age.Seconds()/kvTTL.Seconds())
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

// KeyCountByDomain returns the number of live cached keys per domain.
func (m *EntityMirror) KeyCountByDomain() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[string]int, 16)
	for _, e := range m.entries {
		counts[e.Domain]++
	}
	return counts
}
