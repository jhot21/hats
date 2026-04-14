# HATS

[Home Assistant](https://www.home-assistant.io/) + [NATS](https://nats.io/) = HATS

A bridge that streams Home Assistant events into NATS subjects and exposes a lightweight caching proxy for Home Assistant's REST API.

> **Alpha software.** I have been running this at home for over a year, but breaking changes can occur at any time.

## How it works

HATS connects to Home Assistant via websocket and republishes state-change events onto NATS subjects in real time. Subscriber services connect to NATS and react to events — no polling, no direct Home Assistant dependency in your subscriber code. HATS also runs an HTTP API that caches entity state in a JetStream KV store, so state reads inside event handlers don't hit Home Assistant on every event.

```
Home Assistant ──ws──▶ HATS ──pub──▶ NATS
                         │
                         └──────────▶ HTTP API (cached state, timers, schedules, commands)
                                           ▲
                              Subscriber ──┘
```

## Quick start

**1. Run HATS alongside NATS** (see [`example/compose.yaml`](example/compose.yaml) for a fuller version):

```yaml
services:
  nats:
    image: nats:2
    command: -js  # enable JetStream
    ports:
      - "4222:4222"

  hats:
    image: codeberg.org/jhot/hats:latest
    ports:
      - "8888:8888"
    environment:
      - HASS_HOST=192.168.1.100      # your Home Assistant IP
      - HASS_TOKEN=your-ha-token     # long-lived access token from HA profile
      - NATS_HOST=nats
      - HATS_TOKEN=changeme          # bearer token for the HATS API
    depends_on:
      - nats
```

**2. Write a subscriber:**

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
    cfg, _ := config.New()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    hatsClient := client.NewHatsClient(cfg.GetHatsBaseUrl(), cfg.HatsToken)

    natsClient := nats.New(
        nats.WithoutJetstream,
        nats.WithHostName(cfg.NatsHost),
        nats.WithPort(cfg.NatsPort),
        nats.WithClientName(cfg.NatsClientName),
        nats.WithLogger(logger),
    )
    natsClient.Connect()
    defer natsClient.Close()

    // React when the sun rises
    natsClient.SubscribeState("sun.sun", func(state homeassistant.StateData) error {
        if state.State == "above_horizon" {
            if motionOn, _ := hatsClient.GetStateBool("binary_sensor.living_room_motion"); !motionOn {
                return hatsClient.CallService("light.living_room", homeassistant.Services.TurnOff)
            }
        }
        return nil
    })

    sigch := make(chan os.Signal, 1)
    signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
    <-sigch
}
```

See the [subscriber guide](docs/subscriber-guide.md) for a full walkthrough.

## NATS topics

| Subject | Fires when | Payload |
|---|---|---|
| `homeassistant.states.{domain}.{entity}.{state}` | Entity state changes | `StateData` |
| `homeassistant.attributes.{domain}.{entity}.{state}` | Attributes change, state unchanged | `StateData` |
| `homeassistant.zha.{device_ieee}` | ZHA device event | ZHA event data |
| `homeassistant.zwave-scene.{device_id}` | Z-Wave JS scene event | ZwaveJS value notification |
| `homeassistant.nfc.{tag_id}` | NFC tag scanned | NFC tag data |
| `homeassistant.timers.{name}.finished` | HATS timer expired | `"finished"` |
| `schedules.{name}` | HATS cron schedule fired | `"finished"` |
| `command.{name}` | HTTP request (any method) to `/api/command/{name}` | JSON `CommandPayload` |

See [docs/nats-topics.md](docs/nats-topics.md) for full payload shapes and NATS CLI examples.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `HASS_HOST` | `127.0.0.1` | Home Assistant hostname or IP |
| `HASS_TOKEN` | *(required)* | Long-lived access token from HA profile |
| `NATS_HOST` | `127.0.0.1` | NATS server hostname |
| `NATS_PORT` | `4222` | NATS server port |
| `HATS_TOKEN` | — | Bearer token for the HATS API (optional) |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, or `ERROR` |

See [docs/getting-started.md](docs/getting-started.md) for the full variable reference.

## Features

- Stream Home Assistant state changes, ZHA, Z-Wave, and NFC events to NATS
- Caching proxy for Home Assistant state — fast reads without polling HA
- Server-side countdown timers with NATS notifications on expiry
- Cron-based schedules with second-level precision
- HTTP → NATS command bridge for triggering subscriber logic from webhooks
- Domain-specific NATS subscriber helpers (`SubscribeState`, `SubscribeZHA`, `SubscribeCommand`, etc.)
- Direct Home Assistant REST client with concurrent service calls and history API
- Third-party clients: [ntfy](https://ntfy.sh), [Gokapi](https://github.com/Forceu/Gokapi), [Syncthing](https://github.com/syncthing/syncthing), [National Weather Service](https://www.weather.gov/), [Amcrest cameras](https://amcrest.com/), [Infisical](https://infisical.com/)

## Further reading

- [Getting started](docs/getting-started.md) — installation, environment variables, Infisical secrets
- [Subscriber guide](docs/subscriber-guide.md) — writing a subscriber from scratch
- [HatsClient vs HA RestClient](docs/hats-client-vs-ha-client.md) — when to use each client
- [NATS topics](docs/nats-topics.md) — full topic reference with payload shapes
- [Timers and schedules](docs/timers-and-schedules.md) — server-side timers and cron schedules
- [Commands](docs/commands.md) — HTTP → NATS command bridge
- [Third-party clients](docs/third-party-clients.md) — ntfy, Gokapi, Syncthing, and more
- [API reference](docs/api-reference.md) — HATS HTTP API endpoint reference
