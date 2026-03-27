# HatsClient vs HomeAssistant RestClient

Subscribers have two options for interacting with Home Assistant: the **HatsClient** (`pkg/client`), which talks to the HATS HTTP API, and the **HomeAssistant RestClient** (`pkg/homeassistant`), which talks directly to HA. Choosing the right one keeps your subscriber fast and your HA instance happy.

---

## HatsClient (`pkg/client`)

Talks to the HATS HTTP API at `HATS_HOST:HATS_PORT`. Your subscriber only needs HATS reachable — no direct HA connection required.

```go
import "codeberg.org/jhot/hats/pkg/client"

hatsClient := client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)
```

### State reads

```go
// Returns state as a plain string — served from the JetStream KV cache
state, err := hatsClient.GetState("light.living_room")            // "on" or "off"
on, err    := hatsClient.GetStateBool("binary_sensor.motion")     // true/false
temp, err  := hatsClient.GetStateFloat("sensor.temperature")      // 21.5
state      := hatsClient.GetStateWithDefault("sensor.foo", "unknown") // never errors

// Full StateData including attributes — still cached if available
data, err  := hatsClient.GetStateFull("light.living_room")
brightness := data.Attributes["brightness"]
```

### Service calls

```go
// auto-derives domain from entity ID (light.living_room → "light")
err := hatsClient.CallService("light.living_room", homeassistant.Services.TurnOn)

// with extra service data
err := hatsClient.CallService("light.living_room", homeassistant.Services.TurnOn,
    map[string]any{homeassistant.ExtraProps.BrightnessPercent: 50})

// explicit domain override via "domain" key
err := hatsClient.CallService("scene.movie_night", homeassistant.Services.TurnOn,
    map[string]any{"domain": "scene"})
```

### Timers and schedules

See [timers-and-schedules.md](timers-and-schedules.md) for full details.

```go
hatsClient.StartTimer("motion_off", "10m", false) // start if not running
hatsClient.CancelTimer("motion_off")
hatsClient.SetSchedule("hourly", "0 0 * * * *")
```

### Notifications

```go
// Proxied through HATS — no direct ntfy connection needed in the subscriber
hatsClient.SendNtfyMessage(ntfy.Message{
    Topic:   "alerts",
    Message: "Front door opened",
})
```

See [third-party-clients.md#ntfy](third-party-clients.md#ntfy) for the `ntfy.Message` struct fields.

### Limitations

- `GetState` returns only the state string, not attributes — use `GetStateFull` when you need them
- No history API
- Adds one network hop (subscriber → HATS → HA) for service calls vs calling HA directly

---

## HomeAssistant RestClient (`pkg/homeassistant`)

Talks directly to Home Assistant's REST API at `HASS_HOST:HASS_PORT`. Requires `HASS_HOST` and `HASS_TOKEN` to be set in your subscriber's environment.

```go
import "codeberg.org/jhot/hats/pkg/homeassistant"

haClient := homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
```

### State reads

```go
// Always returns full StateData — no caching
data, err := haClient.GetState("light.living_room")
brightness := data.Attributes["brightness"]
```

### Service calls

```go
// Single service call (auto-derives domain)
err := haClient.CallService("light.living_room", homeassistant.Services.TurnOff)

// Manual domain — needed when domain ≠ entity prefix (e.g. automation.trigger)
err := haClient.CallServiceManual("automation", "automation.my_automation", "trigger")

// Concurrent parallel service calls — waits for all and joins errors
err := haClient.CallServices(
    &homeassistant.CallServiceInput{EntityID: "light.kitchen", Service: homeassistant.Services.TurnOff},
    &homeassistant.CallServiceInput{EntityID: "light.hallway", Service: homeassistant.Services.TurnOff},
    &homeassistant.CallServiceInput{EntityID: "switch.fan",    Service: homeassistant.Services.TurnOff},
)
```

### History API

There is no equivalent in HATS — this is a RestClient exclusive.

```go
history, err := haClient.GetHistory(
    homeassistant.HistoryStartTimestamp(time.Now().Add(-24 * time.Hour)),
    homeassistant.HistoryEndTimestamp(time.Now()),
    homeassistant.HistoryEntityFilter("sensor.temperature"),
    homeassistant.HistorySignificantOnly(),
)

// HistoryResponse is [][]HistoryEntry — one slice per requested entity
onDuration, err := history.GetOnDuration("on", "off")
```

Available option functions: `HistoryStartTimestamp`, `HistoryEndTimestamp`, `HistoryEntityFilter`, `HistoryMinimalResponse`, `HistoryNoAttributes`, `HistorySignificantOnly`.

### Limitations

- Every call hits HA directly — avoid calling in high-frequency event handlers
- Requires `HASS_HOST` + `HASS_TOKEN` in the subscriber's environment
- No timer, schedule, or notification helpers

---

## Decision guide

```
Need entity history data?            → RestClient.GetHistory
Need parallel service calls?         → RestClient.CallServices
Need to trigger an automation?       → RestClient.CallServiceManual("automation", ...)
Reading state in an event handler?   → HatsClient.GetState (cached, fast)
Calling a single service?            → HatsClient.CallService (simpler, one less env var)
Managing timers or schedules?        → HatsClient only — see timers-and-schedules.md
Sending a notification?              → HatsClient.SendNtfyMessage (proxied)
```

### Both together

It's common to use both in the same subscriber. Use HatsClient for the fast path (state reads, service calls, timers) and RestClient only when you specifically need history or concurrent calls:

```go
hatsClient := client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)
haClient   := homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)

// In an hourly schedule handler: fetch history, then act
natsClient.SubscribeSchedule("hourly", func() error {
    history, err := haClient.GetHistory(
        homeassistant.HistoryStartTimestamp(time.Now().Add(-1 * time.Hour)),
        homeassistant.HistoryEntityFilter("binary_sensor.motion"),
    )
    if err != nil {
        return err
    }
    // ... analyze and act using hatsClient.CallService ...
    return nil
})
```

---

← [Subscriber guide](subscriber-guide.md)
