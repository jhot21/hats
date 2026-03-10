package nats

import (
	"encoding/json"
	"testing"
)

func TestCommandPayload_BodyString(t *testing.T) {
	p := CommandPayload{Body: []byte("hello world")}
	if got := p.BodyString(); got != "hello world" {
		t.Errorf("BodyString() = %q, want %q", got, "hello world")
	}
}

func TestCommandPayload_BodyString_Empty(t *testing.T) {
	p := CommandPayload{}
	if got := p.BodyString(); got != "" {
		t.Errorf("BodyString() = %q, want empty string", got)
	}
}

func TestCommandPayload_BodyBytes(t *testing.T) {
	b := []byte{1, 2, 3}
	p := CommandPayload{Body: b}
	if got := p.BodyBytes(); string(got) != string(b) {
		t.Errorf("BodyBytes() mismatch")
	}
}

func TestCommandPayload_BodyJSON(t *testing.T) {
	type Foo struct {
		Name string `json:"name"`
	}
	b, _ := json.Marshal(Foo{Name: "bar"})
	p := CommandPayload{Body: b}

	var result Foo
	if err := p.BodyJSON(&result); err != nil {
		t.Fatalf("BodyJSON() error = %v", err)
	}
	if result.Name != "bar" {
		t.Errorf("BodyJSON() Name = %q, want %q", result.Name, "bar")
	}
}

func TestCommandPayload_BodyJSON_Invalid(t *testing.T) {
	p := CommandPayload{Body: []byte("not json")}
	var v any
	if err := p.BodyJSON(&v); err == nil {
		t.Error("BodyJSON() expected error for invalid JSON, got nil")
	}
}

func TestCommandPayload_JSONRoundTrip(t *testing.T) {
	original := CommandPayload{
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
		Query:   map[string]string{"foo": "bar"},
		Body:    []byte(`{"key":"value"}`),
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded CommandPayload
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Method != original.Method {
		t.Errorf("Method = %q, want %q", decoded.Method, original.Method)
	}
	if decoded.Headers["Content-Type"] != "application/json" {
		t.Errorf("Headers mismatch")
	}
	if decoded.Query["foo"] != "bar" {
		t.Errorf("Query mismatch")
	}
	if string(decoded.Body) != string(original.Body) {
		t.Errorf("Body = %q, want %q", decoded.Body, original.Body)
	}
}
