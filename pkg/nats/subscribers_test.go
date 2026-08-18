package nats

import (
	"encoding/json"
	"testing"

	ha "github.com/jhot21/go-ha-sdk"
	"github.com/nats-io/nats.go"
)

func makeMsg(t *testing.T, v any) *nats.Msg {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return &nats.Msg{Data: b}
}

func TestParseEventData(t *testing.T) {
	event := ha.EventData{EntityID: "light.living_room"}
	msg := makeMsg(t, event)

	got, err := parseEventData(msg)
	if err != nil {
		t.Fatalf("parseEventData() error = %v", err)
	}
	if got.EntityID != "light.living_room" {
		t.Errorf("EntityID = %q, want %q", got.EntityID, "light.living_room")
	}
}

func TestParseStateData(t *testing.T) {
	event := ha.EventData{
		EntityID: "binary_sensor.door",
		NewState: ha.StateData{State: "on"},
		OldState: ha.StateData{State: "off"},
	}
	msg := makeMsg(t, event)

	got, err := parseStateData(msg)
	if err != nil {
		t.Fatalf("parseStateData() error = %v", err)
	}
	if got.State != "on" {
		t.Errorf("State = %q, want %q", got.State, "on")
	}
}

func TestParseCommandPayload(t *testing.T) {
	payload := CommandPayload{
		Method:  "GET",
		Headers: map[string]string{"X-Foo": "bar"},
		Query:   map[string]string{"baz": "qux"},
		Body:    []byte("hello"),
	}
	msg := makeMsg(t, payload)

	got, err := parseCommandPayload(msg)
	if err != nil {
		t.Fatalf("parseCommandPayload() error = %v", err)
	}
	if got.Method != "GET" {
		t.Errorf("Method = %q, want GET", got.Method)
	}
	if got.Headers["X-Foo"] != "bar" {
		t.Error("Headers mismatch")
	}
	if string(got.Body) != "hello" {
		t.Errorf("Body = %q, want hello", got.Body)
	}
}

func TestParseEventData_Invalid(t *testing.T) {
	msg := &nats.Msg{Data: []byte("not json")}
	if _, err := parseEventData(msg); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
