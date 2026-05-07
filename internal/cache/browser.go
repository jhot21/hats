package cache

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	haStatePrefix = "homeassistant.states."
	kvTTL         = 2 * time.Hour
)

// CacheEntry represents a single HA entity in the JetStream KV cache.
type CacheEntry struct {
	Domain    string
	EntityID  string  // e.g. "sensor.thermo1_temperature" (prefix stripped)
	State     string
	UpdatedAt time.Time
	TTLPct    float64 // fraction of 2hr TTL remaining, 0.0–1.0
}

// TTLClass returns the CSS fill class for the TTL bar based on remaining TTL.
func (e CacheEntry) TTLClass() string {
	switch {
	case e.TTLPct > 0.5:
		return "fresh"
	case e.TTLPct > 0.2:
		return "aging"
	default:
		return "old"
	}
}

// ListEntries returns cached HA entities filtered by domain and search query,
// sorted by UpdatedAt descending (most recently changed first).
// domain: exact domain match (empty = all). q: case-insensitive substring match on EntityID (empty = all).
func ListEntries(ctx context.Context, kv jetstream.KeyValue, domain, q string) ([]CacheEntry, error) {
	watcher, err := kv.WatchAll(ctx, jetstream.IgnoreDeletes())
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			return nil, nil
		}
		return nil, err
	}
	defer watcher.Stop()

	now := time.Now()
	q = strings.ToLower(q)
	var entries []CacheEntry

	for entry := range watcher.Updates() {
		if entry == nil {
			break // initial snapshot complete
		}
		key := entry.Key()
		if !strings.HasPrefix(key, haStatePrefix) {
			continue
		}
		entityID := strings.TrimPrefix(key, haStatePrefix)
		entDomain, _, _ := strings.Cut(entityID, ".")

		if domain != "" && entDomain != domain {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(entityID), q) {
			continue
		}

		age := now.Sub(entry.Created())
		ttlPct := max(0, 1.0-age.Seconds()/kvTTL.Seconds())

		entries = append(entries, CacheEntry{
			Domain:    entDomain,
			EntityID:  entityID,
			State:     string(entry.Value()),
			UpdatedAt: entry.Created(),
			TTLPct:    ttlPct,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})
	return entries, nil
}
