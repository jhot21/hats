package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgnats "github.com/jhot21/hats/pkg/nats"
)

func TestCommandHandler_StripsAuthHeaders(t *testing.T) {
	var published pkgnats.CommandPayload

	// Intercept NATS publish to capture the payload
	orig := commandPublish
	commandPublish = func(_ string, data []byte) {
		json.Unmarshal(data, &published) //nolint:errcheck
	}
	defer func() { commandPublish = orig }()

	req := httptest.NewRequest(http.MethodPost, "/api/command/test?foo=bar", strings.NewReader(`{"key":"val"}`))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom", "keep-me")

	w := httptest.NewRecorder()
	postCommandHandler(w, req)

	if _, ok := published.Headers["Authorization"]; ok {
		t.Error("Authorization header should be stripped")
	}
	if _, ok := published.Headers["Cookie"]; ok {
		t.Error("Cookie header should be stripped")
	}
	if published.Headers[http.CanonicalHeaderKey("X-Custom")] == "" {
		t.Error("X-Custom header should be forwarded")
	}
	if published.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", published.Method)
	}
	if published.Query["foo"] != "bar" {
		t.Error("Query param foo should be forwarded")
	}
}
