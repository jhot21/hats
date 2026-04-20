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

	"github.com/go-chi/chi/v5"
)

// RequiredScope returns the scope a request must satisfy based on its route and method.
// Called after chi routing so URL params are available.
func RequiredScope(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "*:*:*"
	}
	pattern := rctx.RoutePattern()
	switch pattern {
	case "/api/state/{entityId}":
		entity := chi.URLParam(r, "entityId")
		if r.Method == http.MethodGet {
			return "read:states:" + entity
		}
		return "write:states:" + entity
	case "/api/state/{entityId}/{service}":
		return "write:states:" + chi.URLParam(r, "entityId")
	case "/api/timer/{timerName}":
		name := chi.URLParam(r, "timerName")
		if r.Method == http.MethodGet {
			return "read:timer:" + name
		}
		return "write:timer:" + name
	case "/api/schedule/{scheduleName}":
		name := chi.URLParam(r, "scheduleName")
		if r.Method == http.MethodGet {
			return "read:schedule:" + name
		}
		return "write:schedule:" + name
	case "/api/ntfy", "/api/ntfy/clear":
		return "write:ntfy:*"
	case "/api/command/{commandName}":
		return "write:command:" + chi.URLParam(r, "commandName")
	}
	return "*:*:*"
}

// HasScope reports whether any scope in granted satisfies the required scope.
// required must be a valid "action:resource:identifier" triplet.
func HasScope(granted []string, required string) bool {
	reqParts := strings.SplitN(required, ":", 3)
	if len(reqParts) != 3 || required == "" {
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
		if strings.HasSuffix(gPart, ".*") {
			prefix := strings.TrimSuffix(gPart, ".*")
			if strings.HasPrefix(req, prefix+".") {
				continue
			}
		}
		return false
	}
	return true
}
