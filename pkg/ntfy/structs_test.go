package ntfy

import (
	"encoding/json"
	"testing"
)

func TestActionHeaders_SerializesAsObject(t *testing.T) {
	msg := Message{
		Topic: "test",
		Title: "Test",
		Actions: []struct {
			Action  string            `json:"action"`
			Label   string            `json:"label"`
			URL     string            `json:"url,omitempty"`
			Method  string            `json:"method,omitempty"`
			Headers map[string]string `json:"headers,omitempty"`
			Body    string            `json:"body,omitempty"`
			Intent  string            `json:"intent,omitempty"`
			Clear   bool              `json:"clear,omitempty"`
			Extras  map[string]string `json:"extras,omitempty"`
		}{
			{
				Action:  "http",
				Label:   "Turn on fan",
				URL:     "https://example.com/command",
				Headers: map[string]string{"Authorization": "Bearer token123"},
				Clear:   true,
			},
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	actions, ok := decoded["actions"].([]any)
	if !ok || len(actions) == 0 {
		t.Fatal("expected non-empty actions array")
	}
	action, ok := actions[0].(map[string]any)
	if !ok {
		t.Fatalf("action should be object, got %T", actions[0])
	}
	headers, ok := action["headers"].(map[string]any)
	if !ok {
		t.Fatalf("headers should be JSON object, got %T: %v", action["headers"], action["headers"])
	}
	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", headers["Authorization"], "Bearer token123")
	}
}
