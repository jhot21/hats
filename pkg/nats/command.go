package nats

import "encoding/json"

// CommandPayload is the NATS message published by the command HTTP endpoint.
// It carries the full HTTP context so subscribers can act on method, headers,
// query parameters, and body without needing HTTP access.
type CommandPayload struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Body    []byte            `json:"body"`
}

// BodyString returns the request body decoded as a UTF-8 string.
func (p CommandPayload) BodyString() string { return string(p.Body) }

// BodyJSON unmarshals the request body as JSON into v.
func (p CommandPayload) BodyJSON(v any) error { return json.Unmarshal(p.Body, v) }

// BodyBytes returns the raw request body bytes.
func (p CommandPayload) BodyBytes() []byte { return p.Body }
