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
		// state (canonical singular)
		{http.MethodGet, "/api/state/light.bedroom", "read:state:light.bedroom"},
		{http.MethodPost, "/api/state/light.bedroom", "write:state:light.bedroom"},
		{http.MethodPost, "/api/state/light.bedroom/turn_on", "write:state:light.bedroom"},

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

		// secret
		{http.MethodGet, "/api/secret/api_key", "read:secret:api_key"},
		{http.MethodPost, "/api/secret/api_key", "write:secret:api_key"},
		{http.MethodPut, "/api/secret/api_key", "write:secret:api_key"},
		{http.MethodDelete, "/api/secret/api_key", "write:secret:api_key"},

		// bulk secrets
		{http.MethodGet, "/api/secrets", "read:secret:*"},
		{http.MethodPost, "/api/secrets", "*:*:*"},

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
		{"full wildcard grants anything", []string{"*:*:*"}, "read:state:light.living_room", true},
		{"full wildcard grants write", []string{"*:*:*"}, "write:command:bedtime", true},

		// exact match
		{"exact match", []string{"read:state:light.living_room"}, "read:state:light.living_room", true},
		{"exact match wrong action", []string{"read:state:light.living_room"}, "write:state:light.living_room", false},
		{"exact match wrong resource", []string{"read:state:light.living_room"}, "read:timer:light.living_room", false},
		{"exact match wrong identifier", []string{"read:state:light.living_room"}, "read:state:light.kitchen", false},

		// wildcard action
		{"wildcard action read", []string{"*:state:light.living_room"}, "read:state:light.living_room", true},
		{"wildcard action write", []string{"*:state:light.living_room"}, "write:state:light.living_room", true},

		// wildcard identifier
		{"wildcard identifier", []string{"read:state:*"}, "read:state:light.living_room", true},
		{"wildcard identifier wrong resource", []string{"read:state:*"}, "read:timer:my_timer", false},

		// prefix wildcard
		{"prefix wildcard matches", []string{"write:state:light.*"}, "write:state:light.living_room", true},
		{"prefix wildcard matches nested", []string{"write:state:light.*"}, "write:state:light.kitchen_ceiling", true},
		{"prefix wildcard no match different domain", []string{"write:state:light.*"}, "write:state:switch.bedroom", false},
		{"prefix wildcard no match partial", []string{"write:state:light.*"}, "write:state:lightbulb", false},

		// wildcard resource
		{"wildcard resource", []string{"read:*:my_thing"}, "read:state:my_thing", true},
		{"wildcard resource and identifier", []string{"read:*:*"}, "read:timer:foo", true},

		// multiple scopes
		{"multiple grants, one matches", []string{"read:state:light.*", "write:command:bedtime"}, "write:command:bedtime", true},
		{"multiple grants, none match", []string{"read:state:*", "read:timer:*"}, "write:state:light.x", false},

		// plural/singular normalization in granted scopes
		{"plural states normalizes", []string{"read:states:light.x"}, "read:state:light.x", true},
		{"plural timers normalizes", []string{"read:timers:my_timer"}, "read:timer:my_timer", true},
		{"plural schedules normalizes", []string{"write:schedules:morning"}, "write:schedule:morning", true},
		{"plural commands normalizes", []string{"write:commands:bedtime"}, "write:command:bedtime", true},
		{"plural secrets normalizes", []string{"read:secrets:api_key"}, "read:secret:api_key", true},
		{"wildcard plural normalizes", []string{"read:states:*"}, "read:state:light.x", true},

		// secret resource
		{"secret read", []string{"read:secret:api_key"}, "read:secret:api_key", true},
		{"secret write", []string{"write:secret:api_key"}, "write:secret:api_key", true},
		{"secret wildcard", []string{"read:secret:*"}, "read:secret:any_key", true},
		{"secret wrong action", []string{"read:secret:api_key"}, "write:secret:api_key", false},

		// empty
		{"no grants", []string{}, "read:state:*", false},
		{"empty required scope", []string{"*:*:*"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tokens.HasScope(tc.granted, tc.required))
		})
	}
}
