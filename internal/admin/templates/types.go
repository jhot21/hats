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

// NavLinkClass returns nav link classes, highlighting the item when title matches section.
func NavLinkClass(title, section string) string {
	if title == section {
		return "flex items-center px-3 py-2 text-sm rounded-md text-slate-100 bg-slate-700 font-medium"
	}
	return "flex items-center px-3 py-2 text-sm rounded-md text-slate-400 hover:text-slate-100 hover:bg-slate-700 transition-colors"
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
