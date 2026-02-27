package util

import (
	"testing"
)

func TestGetEnvWithDefault_EnvSet(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	if got := GetEnvWithDefault("TEST_KEY", "default"); got != "value" {
		t.Fatalf("expected %q, got %q", "value", got)
	}
}

func TestGetEnvWithDefault_EnvNotSet(t *testing.T) {
	if got := GetEnvWithDefault("TEST_KEY_UNSET_XYZ", "default"); got != "default" {
		t.Fatalf("expected %q, got %q", "default", got)
	}
}

func TestGetEnvWithDefault_EnvEmpty(t *testing.T) {
	t.Setenv("TEST_KEY_EMPTY", "")
	if got := GetEnvWithDefault("TEST_KEY_EMPTY", "default"); got != "default" {
		t.Fatalf("expected fallback %q for empty env var, got %q", "default", got)
	}
}

func TestGetEnvWithDefault_DefaultEmpty(t *testing.T) {
	if got := GetEnvWithDefault("TEST_KEY_UNSET_XYZ2", ""); got != "" {
		t.Fatalf("expected %q, got %q", "", got)
	}
}
