# Subscriber guide

This guide walks through building a subscriber service — a Go program that connects to NATS and reacts to Home Assistant events forwarded by HATS.

Prerequisites: HATS and NATS must both be running. See [getting-started.md](getting-started.md) first.

---

## 1. Module setup

```sh
go get codeberg.org/jhot/hats
```

Your subscriber imports from `codeberg.org/jhot/hats/pkg/...`.

---

## 2. Initializing the NATS client

```go
package main

import (
    "log/slog"
    "os"

    "codeberg.org/jhot/hats/pkg/config"
    "codeberg.org/jhot/hats/pkg/nats"
)

func main() {
    cfg, err := config.New()
    if err != nil {
        panic(err)
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: cfg.GetLogLevel(),
    }))

    natsClient := nats.New(
        nats.WithoutJetstream,            // subscribers don't need JetStream
        nats.WithHostName(cfg.NatsHost),
        nats.WithPort(cfg.NatsPort),
        nats.WithClientName(cfg.NatsClientName),
        nats.WithLogger(logger),
    )
    if err := natsClient.Connect(); err != nil {
        panic(err)
    }
    defer natsClient.Close()

    // ... register subscriptions here ...
}
```

For making state reads and service calls inside handlers, see [hats-client-vs-ha-client.md](hats-client-vs-ha-client.md).

---

## 3. Subscription methods

Choose the method that matches what your handler needs from the event:

| Method | Handler signature | Use when |
|---|---|---|
| `AddStateSub` | `func(ha.StateData) error` | You need the new entity state — the most common case |
| `AddEventSub` | `func(ha.EventData) error` | You need old and new state, or for ZHA / Z-Wave / attribute topics |
| `AddSub` | `func() error` | The topic carries no useful payload (timers, schedules, NFC) |
| `AddRawSub` | `func([]byte) error` | You want the raw bytes (commands, custom parsing) |

All methods return `(*nats.Subscription, error)`. Errors are logged automatically; returning a non-nil error from your handler logs it and moves on.

---

## 4. Topic types

Each method takes a `TopicType` constant and an entity ID. The constant determines the NATS subject pattern — HATS constructs the full subject for you.

| Constant | Subject pattern | Typical entity ID | Details |
|---|---|---|---|
| `nats.StateTopicType` | `homeassistant.states.{domain}.{entity}.>` | `sun.sun`, `binary_sensor.motion` | [nats-topics.md](nats-topics.md) |
| `nats.AttributeTopicType` | `homeassistant.attributes.{domain}.{entity}.>` | any entity | [nats-topics.md](nats-topics.md) |
| `nats.TimerTopicType` | `homeassistant.timers.{name}.finished` | timer name | [timers-and-schedules.md](timers-and-schedules.md) |
| `nats.ScheduleTopicType` | `schedules.{name}` | schedule name | [timers-and-schedules.md](timers-and-schedules.md) |
| `nats.CommandTopicType` | `command.{name}` | command name | [commands.md](commands.md) |
| `nats.NfcTopicType` | `homeassistant.nfc.{tag_id}` | NFC tag ID | [nats-topics.md](nats-topics.md) |
| `nats.ZhaTopicType` | `homeassistant.zha.{ieee}` | device IEEE address | [nats-topics.md](nats-topics.md) |
| `nats.ZwaveSceneTopicType` | `homeassistant.zwave-scene.{device_id}` | device ID | [nats-topics.md](nats-topics.md) |

See [nats-topics.md](nats-topics.md) for full payload shapes and NATS CLI examples for each topic.

---

## 5. Full example

This is the annotated version of [`example/example.go`](../example/example.go):

```go
package main

import (
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "codeberg.org/jhot/hats/pkg/client"
    "codeberg.org/jhot/hats/pkg/config"
    "codeberg.org/jhot/hats/pkg/homeassistant"
    "codeberg.org/jhot/hats/pkg/nats"
)

func main() {
    // config.New() reads env vars (and Infisical secrets if configured).
    cfg, err := config.New()
    if err != nil {
        panic(err)
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: cfg.GetLogLevel(),
    }))

    // HatsClient talks to the HATS HTTP API (cached state proxy, timers, schedules).
    // It does NOT need a direct connection to Home Assistant.
    hatsClient := client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)

    // NatsConnection connects to NATS for subscribing to events.
    natsClient := nats.New(
        nats.WithoutJetstream,
        nats.WithHostName(cfg.NatsHost),
        nats.WithPort(cfg.NatsPort),
        nats.WithClientName(cfg.NatsClientName),
        nats.WithLogger(logger),
    )
    if err := natsClient.Connect(); err != nil {
        panic(err)
    }
    defer natsClient.Close()

    // AddStateSub fires whenever sun.sun changes state.
    // The handler receives the new StateData — state.State is the new state string.
    natsClient.AddStateSub(nats.StateTopicType, "sun.sun", func(state homeassistant.StateData) error {
        if state.State == "above_horizon" {
            // GetStateBool reads from HATS's KV cache — fast, no direct HA call.
            if motionOn, _ := hatsClient.GetStateBool("binary_sensor.living_room_motion"); !motionOn {
                logger.Info("Sun above horizon and no motion — turning off living room lights")
                // CallService proxies the request to Home Assistant via HATS.
                return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOff)
            }
        }
        return nil
    })

    // Block until SIGINT or SIGTERM.
    sigch := make(chan os.Signal, 1)
    signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
    <-sigch
    logger.Info("Shutting down")
    // defer natsClient.Close() drains the connection cleanly.
}
```

---

## 6. Common patterns

### Multiple subscriptions

Register as many subscriptions as you need before blocking on the signal channel — they all run concurrently:

```go
natsClient.AddStateSub(nats.StateTopicType, "binary_sensor.front_door", frontDoorHandler)
natsClient.AddStateSub(nats.StateTopicType, "binary_sensor.back_door", backDoorHandler)
natsClient.AddSub(nats.TimerTopicType, "away_mode", awayModeTimerHandler)
natsClient.AddRawSub(nats.CommandTopicType, "bedtime", bedtimeCommandHandler)
```

### Reading state in a handler

Use `HatsClient.GetState` (returns a string) or `GetStateBool` / `GetStateFloat` for typed reads. These are served from the NATS KV cache — no direct HA connection needed in your subscriber:

```go
natsClient.AddStateSub(nats.StateTopicType, "input_boolean.guest_mode", func(state homeassistant.StateData) error {
    if state.State == "on" {
        lights, _ := hatsClient.GetStateBool("light.living_room")
        logger.Info("Guest mode on", "lights_currently_on", lights)
    }
    return nil
})
```

### ZHA button press

ZHA events use `AddEventSub` and `ZhaTopicType`. The handler receives `ha.EventData` with `Command` and `Args` fields:

```go
natsClient.AddEventSub(nats.ZhaTopicType, "00:11:22:33:44:55:66:77", func(event homeassistant.EventData) error {
    switch event.Command {
    case "toggle":
        return hatsClient.CallService("light.bedroom", homeassistant.Services.Toggle)
    case "brightness_move_up":
        return hatsClient.CallService("light.bedroom", homeassistant.Services.TurnOn,
            map[string]any{homeassistant.ExtraProps.BrightnessPercentStep: 10})
    }
    return nil
})
```

### Attribute changes

Attribute events fire when attributes change but the state string stays the same (e.g. a cover's position changes while it stays `open`). Use `AttributeTopicType` with `AddEventSub`:

```go
natsClient.AddEventSub(nats.AttributeTopicType, "cover.blinds", func(event homeassistant.EventData) error {
    pos := event.NewState.Attributes["current_position"]
    logger.Info("Blind position changed", "position", pos)
    return nil
})
```

---

## 7. Graceful shutdown

The pattern above (`signal.Notify` + `defer natsClient.Close()`) gives NATS time to drain in-flight messages before the process exits. Always use `defer natsClient.Close()` rather than calling `Close()` inline.

---

← [Getting started](getting-started.md) · [HatsClient vs HA RestClient →](hats-client-vs-ha-client.md)
