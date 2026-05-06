package templates

import (
	"strings"
	"time"

	"codeberg.org/jhot/hats/internal/cache"
)

// ServiceStatus holds the display state for a single status card.
type ServiceStatus struct {
	Name      string
	Connected bool
	CheckedAt string
}

// TokenData holds display-ready token fields for templates.
type TokenData struct {
	Hash       string
	Name       string
	Scopes     []string
	ExpiresAt  string // "never" or "YYYY-MM-DD"
	LastUsedAt string // "never" or "YYYY-MM-DD HH:MM"
}

// NavLinkClass returns the nav-item CSS class, highlighting the active page.
func NavLinkClass(title, section string) string {
	if title == section {
		return "nav-item active"
	}
	return "nav-item"
}

// ScopeTagClass returns the CSS class for a scope tag.
func ScopeTagClass(scope string) string {
	parts := strings.SplitN(scope, ":", 3)
	switch parts[0] {
	case "write":
		return "scope-tag write"
	case "admin":
		return "scope-tag admin"
	default:
		return "scope-tag"
	}
}

// TokenStatusLabel derives a display status from the token's expiry date.
// Returns "active", "expiring" (within 7 days), or "expired".
func TokenStatusLabel(t TokenData) string {
	if t.ExpiresAt == "never" {
		return "active"
	}
	expiry, err := time.Parse("2006-01-02", t.ExpiresAt)
	if err != nil {
		return "active"
	}
	now := time.Now()
	if expiry.Before(now) {
		return "expired"
	}
	if expiry.Before(now.Add(7 * 24 * time.Hour)) {
		return "expiring"
	}
	return "active"
}

// TokenStatusPillClass returns the CSS class for the status pill.
func TokenStatusPillClass(t TokenData) string {
	switch TokenStatusLabel(t) {
	case "active":
		return "status-pill active"
	case "expiring":
		return "status-pill expiring"
	case "expired":
		return "status-pill expired"
	default:
		return "status-pill active"
	}
}

// ExpiryClass returns a CSS modifier class for expiry urgency.
func ExpiryClass(expiresAt string) string {
	t := TokenData{ExpiresAt: expiresAt}
	switch TokenStatusLabel(t) {
	case "expiring":
		return "warn"
	case "expired":
		return "crit"
	default:
		return ""
	}
}

// TokenHashPreview returns first 8 + "·····" + last 4 chars of a hash.
func TokenHashPreview(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:8] + "·····" + hash[len(hash)-4:]
}

// SecretData holds display-ready secret fields for templates.
type SecretData struct {
	Name      string
	Type      string // "basic_string" | "encrypted_string" | "yaml" | "json"
	CreatedAt string // "YYYY-MM-DD HH:MM"
	UpdatedAt string // "YYYY-MM-DD HH:MM"
}

// SecretTypeLabel returns a human-readable label for a secret type.
func SecretTypeLabel(typ string) string {
	switch typ {
	case "basic_string":
		return "String"
	case "encrypted_string":
		return "Encrypted"
	case "yaml":
		return "YAML"
	case "json":
		return "JSON"
	default:
		return typ
	}
}

// SecretTypeBadgeClass returns the CSS class for the type badge.
func SecretTypeBadgeClass(typ string) string {
	switch typ {
	case "encrypted_string":
		return "type-badge encrypted"
	case "yaml":
		return "type-badge yaml"
	case "json":
		return "type-badge json"
	default:
		return "type-badge"
	}
}

// HAConnectionData holds pre-populated form values for the HA connection page.
type HAConnectionData struct {
	Host   string
	Port   string
	Secure bool
	Error  string // non-empty when a save error should be displayed
}

// NtfyConnectionData holds pre-populated form values for the ntfy connection page.
type NtfyConnectionData struct {
	Host  string
	Error string // non-empty when a save error should be displayed
}

// CacheStatsData holds stats for the cache stats fragment.
type CacheStatsData struct {
	Domains []cache.DomainStat
}

// Global returns the total hits and misses across all domains.
func (c CacheStatsData) Global() (hits, misses int64) {
	for _, d := range c.Domains {
		hits += d.Hits
		misses += d.Misses
	}
	return
}

// HitRate returns the global cache hit rate as a percentage (0–100).
func (c CacheStatsData) HitRate() float64 {
	hits, misses := c.Global()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// RingDashOffset computes the SVG stroke-dashoffset for a 38-radius ring.
func (c CacheStatsData) RingDashOffset() float64 {
	const circumference = 2 * 3.14159 * 38
	return circumference * (1 - c.HitRate()/100)
}

// DomainHitRate returns the hit rate percentage for a single domain.
func DomainHitRate(d cache.DomainStat) float64 {
	total := d.Hits + d.Misses
	if total == 0 {
		return 0
	}
	return float64(d.Hits) / float64(total) * 100
}

// DomainBadgeClass returns the CSS class for a domain badge.
func DomainBadgeClass(domain string) string {
	switch domain {
	case "sensor":
		return "domain-badge sensor"
	case "binary_sensor":
		return "domain-badge binary-sensor"
	case "timer":
		return "domain-badge timer"
	case "input_boolean":
		return "domain-badge input-boolean"
	case "switch":
		return "domain-badge switch"
	case "climate":
		return "domain-badge climate"
	default:
		return "domain-badge other"
	}
}
