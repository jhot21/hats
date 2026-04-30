package tokens_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"codeberg.org/jhot/hats/internal/tokens"
	"github.com/stretchr/testify/assert"
)

func TestRequiredScope(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		// state
		{http.MethodGet, "/api/state/light.bedroom", "read:states:light.bedroom"},
		{http.MethodPost, "/api/state/light.bedroom", "write:states:light.bedroom"},
		{http.MethodPost, "/api/state/light.bedroom/turn_on", "write:states:light.bedroom"},

		// timer
		{http.MethodGet, "/api/timer/my_timer", "read:timer:my_timer"},
		{http.MethodPost, "/api/timer/my_timer", "write:timer:my_timer"},
		{http.MethodDelete, "/api/timer/my_timer", "write:timer:my_timer"},

		// schedule
		{http.MethodGet, "/api/schedule/morning", "read:schedule:morning"},
		{http.MethodPost, "/api/schedule/morning", "write:schedule:morning"},
		{http.MethodDelete, "/api/schedule/morning", "write:schedule:morning"},

		// ntfy
		{http.MethodPost, "/api/ntfy", "write:ntfy:*"},
		{http.MethodDelete, "/api/ntfy", "write:ntfy:*"},
		{http.MethodPut, "/api/ntfy/clear", "write:ntfy:*"},

		// command
		{http.MethodPost, "/api/command/bedtime", "write:command:bedtime"},

		// unknown — fail safe
		{http.MethodGet, "/api/unknown/thing", "*:*:*"},
		{http.MethodGet, "/other/path", "*:*:*"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			assert.Equal(t, tc.want, tokens.RequiredScope(r))
		})
	}
}

func TestHasScope(t *testing.T) {
	cases := []struct {
		name     string
		granted  []string
		required string
		want     bool
	}{
		// full wildcard
		{"full wildcard grants anything", []string{"*:*:*"}, "read:states:light.living_room", true},
		{"full wildcard grants write", []string{"*:*:*"}, "write:command:bedtime", true},

		// exact match
		{"exact match", []string{"read:states:light.living_room"}, "read:states:light.living_room", true},
		{"exact match wrong action", []string{"read:states:light.living_room"}, "write:states:light.living_room", false},
		{"exact match wrong resource", []string{"read:states:light.living_room"}, "read:timer:light.living_room", false},
		{"exact match wrong identifier", []string{"read:states:light.living_room"}, "read:states:light.kitchen", false},

		// wildcard action
		{"wildcard action read", []string{"*:states:light.living_room"}, "read:states:light.living_room", true},
		{"wildcard action write", []string{"*:states:light.living_room"}, "write:states:light.living_room", true},

		// wildcard identifier
		{"wildcard identifier", []string{"read:states:*"}, "read:states:light.living_room", true},
		{"wildcard identifier wrong resource", []string{"read:states:*"}, "read:timer:my_timer", false},

		// prefix wildcard
		{"prefix wildcard matches", []string{"write:states:light.*"}, "write:states:light.living_room", true},
		{"prefix wildcard matches nested", []string{"write:states:light.*"}, "write:states:light.kitchen_ceiling", true},
		{"prefix wildcard no match different domain", []string{"write:states:light.*"}, "write:states:switch.bedroom", false},
		{"prefix wildcard no match partial", []string{"write:states:light.*"}, "write:states:lightbulb", false},

		// wildcard resource
		{"wildcard resource", []string{"read:*:my_thing"}, "read:states:my_thing", true},
		{"wildcard resource and identifier", []string{"read:*:*"}, "read:timer:foo", true},

		// multiple scopes — any match wins
		{"multiple grants, one matches", []string{"read:states:light.*", "write:command:bedtime"}, "write:command:bedtime", true},
		{"multiple grants, none match", []string{"read:states:*", "read:timer:*"}, "write:states:light.x", false},

		// empty
		{"no grants", []string{}, "read:states:*", false},
		{"empty required scope", []string{"*:*:*"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tokens.HasScope(tc.granted, tc.required))
		})
	}
}
