package templates

import "strings"

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

// ScopeBadgeClass returns the Tailwind CSS classes for a scope badge based on its action.
func ScopeBadgeClass(scope string) string {
	parts := strings.SplitN(scope, ":", 3)
	if len(parts) == 0 {
		return "bg-slate-700 text-slate-300"
	}
	switch parts[0] {
	case "read":
		return "bg-teal-900 text-teal-300"
	case "write":
		return "bg-amber-900 text-amber-300"
	default: // "*" or unknown
		return "bg-purple-900 text-purple-300"
	}
}
