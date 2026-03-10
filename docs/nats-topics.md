# NATS topics

This is the full topic reference for events published by HATS. To subscribe to these topics in Go, see the [subscriber guide](subscriber-guide.md).

You can inspect live traffic with the NATS CLI:

```sh
# Subscribe to all HATS events
nats sub "homeassistant.>"

# Subscribe to everything (including schedules and commands)
nats sub ">"
```

---

## State changes

**Subject:** `homeassistant.states.{domain}.{entity_id}.{state}`

Fires whenever a Home Assistant entity changes state. The `{state}` segment in the subject is the new state value with special characters substituted:

| Character | Replaced with |
|---|---|
| ` ` (space) | `_` |
| `*` | `_` |
| `>` | `_` |
| `.` | `_` |

**Payload:** JSON-encoded `EventData`

```json
{
  "entity_id": "binary_sensor.front_door",
  "new_state": {
    "state": "on",
    "attributes": { "device_class": "door", "friendly_name": "Front door" },
    "last_changed": "2024-01-15T08:30:00Z",
    "last_updated": "2024-01-15T08:30:00Z"
  },
  "old_state": {
    "state": "off",
    "attributes": { "device_class": "door", "friendly_name": "Front door" },
    "last_changed": "2024-01-15T07:00:00Z",
    "last_updated": "2024-01-15T07:00:00Z"
  }
}
```

**Subscribe with:** `SubscribeState` (receives `NewState` only) or `SubscribeAttributes` (receives both old and new state).

```go
// Most common — just the new state
natsClient.SubscribeState("binary_sensor.front_door", func(state ha.StateData) error {
    if state.State == "on" { /* door opened */ }
    return nil
})

// When you need to compare old vs new
natsClient.SubscribeAttributes("binary_sensor.front_door", func(event ha.EventData) error {
    logger.Info("State changed", "from", event.OldState.State, "to", event.NewState.State)
    return nil
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.states.binary_sensor.front_door.>"
nats sub "homeassistant.states.light.>"   # all lights
nats sub "homeassistant.states.>"         # all state changes
```

---

## Attribute changes

**Subject:** `homeassistant.attributes.{domain}.{entity_id}.{state}`

Fires when an entity's attributes change but the state string stays the same. Common for covers (position changes while staying `open`), media players (volume changes while staying `playing`), and climate entities.

**Payload:** JSON-encoded `EventData` (same structure as state changes)

**Subscribe with:** `SubscribeAttributes`.

```go
natsClient.SubscribeAttributes("cover.office_blinds", func(event ha.EventData) error {
    pos := event.NewState.Attributes["current_position"]
    logger.Info("Blind position", "position", pos)
    return nil
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.attributes.cover.office_blinds.>"
nats sub "homeassistant.attributes.>"   # all attribute events
```

---

## ZHA events

**Subject:** `homeassistant.zha.{device_ieee}`

Fires when a ZHA (Zigbee Home Automation) device sends an event — button presses, dial turns, cube gestures, etc. The `{device_ieee}` is the device's IEEE address (e.g. `00:11:22:33:44:55:66:77`).

**Payload:** JSON-encoded `EventData` with ZHA-specific fields:

```json
{
  "device_ieee": "00:11:22:33:44:55:66:77",
  "device_id": "abc123",
  "command": "toggle",
  "args": { "press_type": "short_release" },
  "params": {}
}
```

**Subscribe with:** `SubscribeZHA`.

```go
natsClient.SubscribeZHA("00:11:22:33:44:55:66:77", func(event ha.EventData) error {
    switch event.Command {
    case "toggle":
        return hatsClient.CallService("light.bedroom", ha.Services.Toggle)
    case "brightness_move_up":
        return hatsClient.CallService("light.bedroom", ha.Services.TurnOn,
            map[string]any{ha.ExtraProps.BrightnessPercentStep: 10})
    }
    return nil
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.zha.00:11:22:33:44:55:66:77"
nats sub "homeassistant.zha.>"   # all ZHA events
```

---

## Z-Wave JS scene events

**Subject:** `homeassistant.zwave-scene.{device_id}`

Fires when a Z-Wave JS device sends a scene/value notification — typically button presses on Z-Wave remotes or keypads. The `{device_id}` is the Z-Wave node's device ID string from Home Assistant.

**Payload:** JSON-encoded `EventData` with Z-Wave fields:

```json
{
  "domain": "zwave_js",
  "node_id": 42,
  "endpoint": 0,
  "command_class": 91,
  "command_class_name": "Central Scene",
  "label": "Scene 001",
  "property": "scene",
  "property_name": "scene",
  "property_key": "001",
  "property_key_name": "001",
  "value": "KeyPressed",
  "value_raw": 0
}
```

See the [Home Assistant Z-Wave JS integration docs](https://www.home-assistant.io/integrations/zwave_js/#scene-events-value-notification) for field descriptions.

**Subscribe with:** `SubscribeZwaveScene`.

```go
natsClient.SubscribeZwaveScene("my-zwave-device-id", func(event ha.EventData) error {
    if event.PropertyKeyName == "001" && event.Value == "KeyPressed" {
        return hatsClient.CallService("light.hallway", ha.Services.Toggle)
    }
    return nil
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.zwave-scene.my-device-id"
nats sub "homeassistant.zwave-scene.>"
```

---

## NFC tag scanned

**Subject:** `homeassistant.nfc.{tag_id}`

Fires when an NFC tag is scanned by a Home Assistant companion app. The `{tag_id}` is the tag's ID string as configured in HA.

**Payload:** JSON-encoded `EventData` with:

```json
{
  "tag_id": "my-nfc-tag-id"
}
```

**Subscribe with:** `SubscribeNFC`.

```go
natsClient.SubscribeNFC("my-nfc-tag-id", func(event ha.EventData) error {
    return hatsClient.CallService("lock.front_door", ha.Services.Unlock)
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.nfc.my-nfc-tag-id"
nats sub "homeassistant.nfc.>"
```

---

## Timers

**Subject:** `homeassistant.timers.{name}.finished`

Fires when a HATS timer expires. See [timers-and-schedules.md](timers-and-schedules.md) for how to create and manage timers.

**Payload:** `"finished"` (plain text)

**Subscribe with:** `SubscribeTimer`.

```go
natsClient.SubscribeTimer("motion_off", func() error {
    return hatsClient.CallService("light.living_room", ha.Services.TurnOff)
})
```

**NATS CLI:**

```sh
nats sub "homeassistant.timers.motion_off.finished"
nats sub "homeassistant.timers.>"
```

---

## Schedules

**Subject:** `schedules.{name}`

Fires when a HATS cron schedule triggers. See [timers-and-schedules.md](timers-and-schedules.md) for how to create schedules.

**Payload:** `"finished"` (plain text)

**Subscribe with:** `SubscribeSchedule`.

```go
natsClient.SubscribeSchedule("hourly", func() error {
    logger.Info("Hourly tick")
    return nil
})
```

**NATS CLI:**

```sh
nats sub "schedules.hourly"
nats sub "schedules.>"
```

---

## Commands

**Subject:** `command.{name}`

Fires when an HTTP request (any method) is made to `/api/command/{name}` on the HATS API. The request is forwarded as a structured JSON payload. See [commands.md](commands.md) for details.

**Payload:** JSON-encoded `CommandPayload` with HTTP method, headers, query params, and body.

**Subscribe with:** `SubscribeCommand`.

```go
natsClient.SubscribeCommand("bedtime", func(payload nats.CommandPayload) error {
    // payload.Method — the HTTP method used (GET, POST, etc.)
    // payload.BodyString() — body as string
    // payload.BodyJSON(&v) — unmarshal body as JSON
    // payload.BodyBytes() — raw body bytes
    return runBedtimeRoutine()
})
```

**NATS CLI:**

```sh
nats sub "command.bedtime"
nats sub "command.>"
```

---

← [README](../README.md) · [Subscriber guide](subscriber-guide.md)
