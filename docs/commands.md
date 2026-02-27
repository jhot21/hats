# Commands

The command bridge lets external services — webhooks, Home Assistant automations, scripts, or other apps — trigger subscriber logic via HTTP without needing a direct NATS connection.

**How it works:** An HTTP POST to `/api/command/{name}` on the HATS API publishes the raw POST body to the NATS subject `command.{name}`. Any subscriber listening on that subject receives the message immediately.

```
POST /api/command/bedtime   →   NATS: command.bedtime   →   subscriber handler
```

---

## Subscriber side

Commands are registered using `AddRawSub` — see [subscriber-guide.md#subscription-methods](subscriber-guide.md#3-subscription-methods) for the full method table.

```go
natsClient.AddRawSub(nats.CommandTopicType, "bedtime", func(body []byte) error {
    // body contains the raw HTTP POST body bytes
    // It can be empty, plain text, JSON, or any other format
    return runBedtimeRoutine()
})
```

If your command sends a JSON payload, decode it in the handler:

```go
type SonarrPayload struct {
    SeriesTitle string `json:"seriesTitle"`
    EpisodeTitle string `json:"episodeTitle"`
}

natsClient.AddRawSub(nats.CommandTopicType, "sonarr", func(body []byte) error {
    var payload SonarrPayload
    if err := json.Unmarshal(body, &payload); err != nil {
        return err
    }
    return hatsClient.SendNtfyMessage(ntfy.Message{
        Topic:   "media",
        Message: fmt.Sprintf("Downloaded: %s — %s", payload.SeriesTitle, payload.EpisodeTitle),
    })
})
```

---

## Sending a command

**From curl:**

```sh
# Empty body
curl -X POST http://hats:8888/api/command/bedtime \
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
