# Commands

The command bridge lets external services — webhooks, Home Assistant automations, scripts, or other apps — trigger subscriber logic via HTTP without needing a direct NATS connection.

**How it works:** An HTTP request (any method) to `/api/command/{name}` on the HATS API publishes a structured JSON payload to the NATS subject `command.{name}`. Any subscriber listening on that subject receives the message immediately.

```
ANY /api/command/bedtime   →   NATS: command.bedtime   →   subscriber handler
```

---

## Subscriber side

Commands are registered using `SubscribeCommand` — see [subscriber-guide.md#subscription-methods](subscriber-guide.md#3-subscription-methods) for the full method table.

```go
natsClient.SubscribeCommand("bedtime", func(payload nats.CommandPayload) error {
    // payload.Method — the HTTP method used (GET, POST, etc.)
    // payload.BodyString() — body as string
    // payload.BodyJSON(&v) — unmarshal body as JSON
    // payload.BodyBytes() — raw body bytes
    return runBedtimeRoutine()
})
```

If your command sends a JSON payload, use `BodyJSON` to decode it:

```go
type SonarrPayload struct {
    SeriesTitle  string `json:"seriesTitle"`
    EpisodeTitle string `json:"episodeTitle"`
}

natsClient.SubscribeCommand("sonarr", func(payload nats.CommandPayload) error {
    var sonarrPayload SonarrPayload
    if err := payload.BodyJSON(&sonarrPayload); err != nil {
        return err
    }
    return hatsClient.SendNtfyMessage(ntfy.Message{
        Topic:   "media",
        Message: fmt.Sprintf("Downloaded: %s — %s", sonarrPayload.SeriesTitle, sonarrPayload.EpisodeTitle),
    })
})
```

---

## Sending a command

**From curl:**

Any HTTP method is accepted — the method is forwarded in the payload so subscribers can act on it if needed.

```sh
# Empty body (any method works — POST shown here)
curl -X POST http://hats:8888/api/command/bedtime \
  -H "Authorization: Bearer changeme"

# GET with query params
curl http://hats:8888/api/command/bedtime?mode=away \
  -H "Authorization: Bearer changeme"

# JSON body
curl -X POST http://hats:8888/api/command/sonarr \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"seriesTitle":"Fallout","episodeTitle":"The End"}'
```

**From a Home Assistant automation:**

Use the `rest_command` integration to POST to HATS from an automation:

```yaml
# configuration.yaml
rest_command:
  hats_bedtime:
    url: "http://192.168.1.x:8888/api/command/bedtime"
    method: POST
    headers:
      Authorization: "Bearer changeme"
```

Then call `rest_command.hats_bedtime` from any automation action.

**From another Go service:**

```go
resp, err := http.Post(
    "http://hats:8888/api/command/bedtime",
    "application/json",
    strings.NewReader(`{"source":"automation"}`),
)
```

---

## Authentication

All `/api/*` routes require authentication when `HATS_TOKEN` is set. Commands are no exception. See [api-reference.md](api-reference.md) for accepted auth schemes (Bearer token or HTTP Basic Auth).

If `HATS_TOKEN` is empty, all requests are allowed without auth.

---

← [Timers and schedules](timers-and-schedules.md) · [API reference →](api-reference.md)
