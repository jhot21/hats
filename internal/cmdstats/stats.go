package cmdstats

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type cmdTokenKey struct {
	CommandName string
	TokenHash   string
}

type cmdRecord struct {
	mu         sync.Mutex
	tokenName  string
	lastCalled time.Time
	hours      map[string]int64 // YYYYMMDDHH → count
}

// CommandStat holds per-(command,token) stats for the last 24 hours.
type CommandStat struct {
	CommandName string
	TokenHash   string
	TokenName   string
	LastCalled  time.Time
	Hours       [24]int64 // index 0 = oldest hour, index 23 = current hour
}

// Summary holds aggregate stats across all endpoints.
type Summary struct {
	TotalCalls int64
	Endpoints  int
	TokensUsed int
}

// Recorder tracks command endpoint call stats in memory and flushes to KV.
// Pass nil for kv to skip all KV I/O (useful in tests).
type Recorder struct {
	kv      jetstream.KeyValue
	logger  *slog.Logger
	records sync.Map // cmdTokenKey → *cmdRecord
}

// New creates a Recorder backed by the given KV bucket.
func New(kv jetstream.KeyValue, logger *slog.Logger) *Recorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Recorder{kv: kv, logger: logger}
}

// Record increments the current-hour counter for the given command+token pair.
func (r *Recorder) Record(commandName, tokenHash, tokenName string) {
	key := cmdTokenKey{CommandName: commandName, TokenHash: tokenHash}
	v, _ := r.records.LoadOrStore(key, &cmdRecord{hours: make(map[string]int64)})
	rec := v.(*cmdRecord)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.tokenName = tokenName
	rec.lastCalled = time.Now().UTC()
	rec.hours[currentHour()]++
}

// Snapshot returns current stats sorted by 24h call count descending, plus an aggregate summary.
func (r *Recorder) Snapshot() ([]CommandStat, Summary) {
	window := hourWindow(24)

	endpointSeen := make(map[string]bool)
	tokenSeen := make(map[string]bool)
	var stats []CommandStat
	var totalCalls int64

	r.records.Range(func(k, v any) bool {
		key := k.(cmdTokenKey)
		rec := v.(*cmdRecord)
		rec.mu.Lock()
		defer rec.mu.Unlock()

		var hoursArr [24]int64
		for i, h := range window {
			hoursArr[i] = rec.hours[h]
		}
		s := sum24h(hoursArr)
		totalCalls += s
		endpointSeen[key.CommandName] = true
		tokenSeen[key.TokenHash] = true
		stats = append(stats, CommandStat{
			CommandName: key.CommandName,
			TokenHash:   key.TokenHash,
			TokenName:   rec.tokenName,
			LastCalled:  rec.lastCalled,
			Hours:       hoursArr,
		})
		return true
	})

	sort.Slice(stats, func(i, j int) bool {
		si := sum24h(stats[i].Hours)
		sj := sum24h(stats[j].Hours)
		if si != sj {
			return si > sj
		}
		return stats[i].CommandName < stats[j].CommandName
	})

	return stats, Summary{
		TotalCalls: totalCalls,
		Endpoints:  len(endpointSeen),
		TokensUsed: len(tokenSeen),
	}
}

// Load reads existing KV state into memory. Call once at startup before StartFlusher.
func (r *Recorder) Load(ctx context.Context) error {
	if r.kv == nil {
		r.logger.Info("cmdstats: no KV bucket configured, skipping load")
		return nil
	}
	keys, err := r.kv.Keys(ctx)
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			r.logger.Info("cmdstats: KV bucket is empty, nothing to load")
			return nil
		}
		return err
	}
	var loaded int
	for _, key := range keys {
		entry, err := r.kv.Get(ctx, key)
		if err != nil {
			continue
		}
		if cmd, tokenHash, ok := parseMetaKey(key); ok {
			t, err := time.Parse(time.RFC3339, string(entry.Value()))
			if err != nil {
				continue
			}
			k := cmdTokenKey{CommandName: cmd, TokenHash: tokenHash}
			v, _ := r.records.LoadOrStore(k, &cmdRecord{hours: make(map[string]int64)})
			rec := v.(*cmdRecord)
			rec.mu.Lock()
			if rec.lastCalled.IsZero() || t.After(rec.lastCalled) {
				rec.lastCalled = t
			}
			rec.mu.Unlock()
			loaded++
			continue
		}
		if cmd, hour, tokenHash, ok := parseHourlyKey(key); ok {
			n, err := strconv.ParseInt(string(entry.Value()), 10, 64)
			if err != nil || n == 0 {
				continue
			}
			k := cmdTokenKey{CommandName: cmd, TokenHash: tokenHash}
			v, _ := r.records.LoadOrStore(k, &cmdRecord{hours: make(map[string]int64)})
			rec := v.(*cmdRecord)
			rec.mu.Lock()
			rec.hours[hour] = n
			rec.mu.Unlock()
			loaded++
		}
	}
	r.logger.Info("cmdstats: loaded from KV", "keys", len(keys), "records", loaded)
	return nil
}

// StartFlusher starts a background goroutine that writes counters to KV every 60s.
// The first flush happens immediately, then every 60s thereafter.
func (r *Recorder) StartFlusher(ctx context.Context) {
	go func() {
		r.flush(ctx)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.flush(ctx)
			}
		}
	}()
}

// Flush writes all current counters to KV. Call on shutdown to persist data.
func (r *Recorder) Flush(ctx context.Context) {
	r.flush(ctx)
}

func (r *Recorder) flush(ctx context.Context) {
	if r.kv == nil {
		return
	}
	h := currentHour()
	cutoff := time.Now().UTC().Add(-25 * time.Hour).Format("2006010215")
	var recordCount int
	r.records.Range(func(k, v any) bool {
		key := k.(cmdTokenKey)
		rec := v.(*cmdRecord)
		rec.mu.Lock()
		count := rec.hours[h]
		lastCalled := rec.lastCalled
		for hr := range rec.hours {
			if hr < cutoff {
				delete(rec.hours, hr)
			}
		}
		rec.mu.Unlock()

		if count > 0 {
			if _, err := r.kv.PutString(ctx, hourlyKey(key.CommandName, h, key.TokenHash), strconv.FormatInt(count, 10)); err != nil {
				r.logger.Error("cmdstats: failed to write hourly key", "key", hourlyKey(key.CommandName, h, key.TokenHash), "error", err)
			}
		}
		if !lastCalled.IsZero() {
			if _, err := r.kv.PutString(ctx, metaKey(key.CommandName, key.TokenHash), lastCalled.Format(time.RFC3339)); err != nil {
				r.logger.Error("cmdstats: failed to write meta key", "key", metaKey(key.CommandName, key.TokenHash), "error", err)
			}
		}
		recordCount++
		return true
	})
	if recordCount > 0 {
		r.logger.Debug("cmdstats: flushed to KV", "records", recordCount, "hour", h)
	}
}

// --- helpers ---

func currentHour() string {
	return time.Now().UTC().Format("2006010215")
}

func hourWindow(n int) []string {
	now := time.Now().UTC().Truncate(time.Hour)
	hours := make([]string, n)
	for i := range hours {
		hours[n-1-i] = now.Add(-time.Duration(i) * time.Hour).Format("2006010215")
	}
	return hours
}

func sum24h(hours [24]int64) int64 {
	var total int64
	for _, v := range hours {
		total += v
	}
	return total
}

func hourlyKey(cmd, hour, tokenHash string) string {
	return cmd + "." + hour + "." + tokenHash
}

func metaKey(cmd, tokenHash string) string {
	return cmd + ".meta." + tokenHash
}

// parseHourlyKey parses "{cmd}.{YYYYMMDDHH}.{64-char-hash}".
// Works even when cmd itself contains dots.
func parseHourlyKey(key string) (cmd, hour, tokenHash string, ok bool) {
	lastDot := strings.LastIndex(key, ".")
	if lastDot < 0 {
		return
	}
	tokenHash = key[lastDot+1:]
	if len(tokenHash) != 64 {
		return
	}
	rest := key[:lastDot]
	secondDot := strings.LastIndex(rest, ".")
	if secondDot < 0 {
		return
	}
	hour = rest[secondDot+1:]
	if len(hour) != 10 {
		return
	}
	cmd = rest[:secondDot]
	ok = true
	return
}

// parseMetaKey parses "{cmd}.meta.{64-char-hash}".
func parseMetaKey(key string) (cmd, tokenHash string, ok bool) {
	lastDot := strings.LastIndex(key, ".")
	if lastDot < 0 {
		return
	}
	tokenHash = key[lastDot+1:]
	if len(tokenHash) != 64 {
		return
	}
	rest := key[:lastDot]
	if !strings.HasSuffix(rest, ".meta") {
		return
	}
	cmd = rest[:len(rest)-5]
	ok = true
	return
}
