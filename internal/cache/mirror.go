package cache

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jhot21/hats/internal/nats"
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
	var (
		pending      map[string]CacheEntry
		snapshotDone bool
	)

	nats.RunWatch(ctx, m.logger, "entity-mirror",
		func() (jetstream.KeyWatcher, error) {
			return m.kv.WatchAll(ctx)
		},
		func() {
			// Reset snapshot state for each new watcher session.
			pending = make(map[string]CacheEntry)
			snapshotDone = false
		},
		func(entry jetstream.KeyValueEntry) {
			if entry == nil {
				// Snapshot complete — atomically swap the pending map into m.entries.
				// Reads continue to see the old map until this point.
				m.mu.Lock()
				m.entries = pending
				m.mu.Unlock()
				snapshotDone = true
				pending = nil
				return
			}
			m.applyEntry(entry, pending, snapshotDone)
		},
	)
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

// Get returns the cached state string for entityID.
// Returns ("", false) if the entity is not present in the mirror.
func (m *EntityMirror) Get(entityID string) (string, bool) {
	m.mu.RLock()
	e, ok := m.entries[entityID]
	m.mu.RUnlock()
	return e.State, ok
}
