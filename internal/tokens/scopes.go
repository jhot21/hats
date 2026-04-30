// Package tokens implements scope-based access control for HATS API tokens.
//
// # Scope Format
//
// Scopes use the format: {action}:{resource}:{identifier}
//
// Actions:
//   - read  — read-only access
//   - write — write/mutate access
//   - *     — read and write
//
// Resources:
//   - states   — Home Assistant entity states
//   - command  — NATS command publishing
//   - timer    — timer management
//   - schedule — cron schedule management
//   - ntfy     — push notification sending
//   - *        — all resources
//
// Identifiers: the entity ID, command name, timer name, etc.
// Wildcards are supported at any position:
//   - *          matches any value (e.g. read:states:* — read any state)
//   - prefix.*   matches any value with that prefix (e.g. write:states:light.*)
//
// Examples:
//   - *:*:*                    full access
//   - *:states:*               read + write any state
//   - read:states:*            read any state
//   - write:states:light.*     write any light entity
//   - write:states:light.living_room  write exactly that entity
//   - *:command:bedtime        read + write the bedtime command
//
// A token's scopes are a set — a request passes if any scope satisfies the required scope.
package tokens

import (
	"net/http"
	"strings"
)

// RequiredScope returns the scope a request must satisfy based on its URL path and method.
// Parses r.URL.Path directly rather than chi's RoutePattern, because sub-router middleware
// runs before chi resolves the specific sub-route, making RoutePattern unreliable here.
func RequiredScope(r *http.Request) string {
	// Strip the /api/ prefix to get the resource segment(s).
	sub := strings.TrimPrefix(r.URL.Path, "/api/")
	if sub == r.URL.Path {
		// Not an /api/ path — fail safe.
		return "*:*:*"
	}

	// Split into at most 3 parts: resource, identifier, optional extra segment.
	parts := strings.SplitN(sub, "/", 3)
	if len(parts) == 0 || parts[0] == "" {
		return "*:*:*"
	}

	switch parts[0] {
	case "state":
		if len(parts) < 2 || parts[1] == "" {
			break
		}
		if len(parts) == 3 {
			// /api/state/{entityId}/{service}
			return "write:states:" + parts[1]
		}
		// /api/state/{entityId}
		if r.Method == http.MethodGet {
			return "read:states:" + parts[1]
		}
		return "write:states:" + parts[1]
	case "timer":
		if len(parts) < 2 || parts[1] == "" {
			break
		}
		if r.Method == http.MethodGet {
			return "read:timer:" + parts[1]
		}
		return "write:timer:" + parts[1]
	case "schedule":
		if len(parts) < 2 || parts[1] == "" {
			break
		}
		if r.Method == http.MethodGet {
			return "read:schedule:" + parts[1]
		}
		return "write:schedule:" + parts[1]
	case "ntfy":
		return "write:ntfy:*"
	case "command":
		if len(parts) < 2 || parts[1] == "" {
			break
		}
		return "write:command:" + parts[1]
	}
	// Unknown routes require full access — fail safe.
	return "*:*:*"
}

// HasScope reports whether any scope in granted satisfies the required scope.
// required must be a valid "action:resource:identifier" triplet.
func HasScope(granted []string, required string) bool {
	if required == "" {
		return false
	}
	reqParts := strings.SplitN(required, ":", 3)
	if len(reqParts) != 3 {
		return false
	}
	for _, g := range granted {
		if matchScope(g, reqParts) {
			return true
		}
	}
	return false
}

func matchScope(granted string, reqParts []string) bool {
	gParts := strings.SplitN(granted, ":", 3)
	if len(gParts) != 3 {
		return false
	}
	for i, gPart := range gParts {
		if gPart == "*" {
			continue
		}
		req := reqParts[i]
		if gPart == req {
			continue
		}
		// prefix wildcard: "light.*" matches "light.living_room"
		if prefix, ok := strings.CutSuffix(gPart, ".*"); ok {
			if strings.HasPrefix(req, prefix+".") {
				continue
			}
		}
		return false
	}
	return true
}
