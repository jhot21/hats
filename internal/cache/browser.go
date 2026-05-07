package cache

import (
	"time"
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
