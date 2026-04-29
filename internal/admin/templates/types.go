package templates

import (
	"strings"
	"time"
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
