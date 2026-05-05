# Third-party clients

HATS includes client packages for several third-party services. These are available to use in your subscriber as Go packages — they share the same `config.HatsConfig` for environment-based configuration.

Config variables for each service are listed in [getting-started.md#third-party-integrations](getting-started.md#third-party-integrations).

---

## ntfy

Package: `codeberg.org/jhot/hats/pkg/ntfy`

[ntfy](https://ntfy.sh) is an HTTP-based push notification service.

```go
import "codeberg.org/jhot/hats/pkg/ntfy"

client := ntfy.New(cfg.NtfyHost, cfg.NtfyToken)

err := client.Send(ntfy.Message{
    Topic:    "alerts",
    Title:    "Motion detected",
    Message:  "Front door motion at 10:32 PM",
    Priority: ntfy.PriorityHigh,
    Tags:     []string{"rotating_light"},
})
```

**Config:** `NTFY_HOST` (default: `https://ntfy.sh`), `NTFY_TOKEN` (only needed for self-hosted ntfy with auth)

**Via HatsClient (simpler):** If your subscriber already has a HatsClient, you can send ntfy messages through HATS without needing direct ntfy access in the subscriber:

```go
err := hatsClient.SendNtfyMessage(ntfy.Message{
    Topic:   "alerts",
    Message: "Front door opened",
})
```

See [hats-client-vs-ha-client.md](hats-client-vs-ha-client.md) for guidance on which approach to choose.

---

## Gokapi

Package: `codeberg.org/jhot/hats/pkg/gokapi`

[Gokapi](https://github.com/Forceu/Gokapi) is a self-hosted file sharing service. Useful for generating temporary download links from your subscriber.

```go
import "codeberg.org/jhot/hats/pkg/gokapi"

client := gokapi.New(cfg.GokapiHost, cfg.GokapiToken)
```

**Config:** `GOKAPI_HOST` (default: `http://gokapi:53842`), `GOKAPI_TOKEN`

---

## Syncthing

Package: `codeberg.org/jhot/hats/pkg/syncthing`

[Syncthing](https://github.com/syncthing/syncthing) is a peer-to-peer file sync application.

```go
import "codeberg.org/jhot/hats/pkg/syncthing"

client := syncthing.New(cfg.SyncthingHost, cfg.SyncthingToken)
```

**Config:** `SYNCTHING_HOST` (default: `http://127.0.0.1:8384`), `SYNCTHING_TOKEN`

---

## National Weather Service (NWS)

Package: `codeberg.org/jhot/hats/pkg/nws`

Client for the [National Weather Service public API](https://www.weather.gov/) — no API key required. Useful for precipitation totals, forecasts, and temperature data.

Typically initialized using your home zone's coordinates from Home Assistant:

```go
import "codeberg.org/jhot/hats/pkg/nws"

// Get lat/lon from the HA zone state
homeZone, err := haClient.GetState("zone.home")
lat := homeZone.Attributes["latitude"].(float64)
lon := homeZone.Attributes["longitude"].(float64)

weather, err := nws.New(lat, lon)
err = weather.UpdateTemps(21) // fetch forecasted high/low for next N days
```

**Config:** none — uses the public NWS API

---

## Amcrest cameras

Package: `codeberg.org/jhot/hats/pkg/amcrest`

Client for [Amcrest IP cameras](https://amcrest.com/). Uses custom settings (not standard config fields) to support multiple cameras with different credentials:

```go
import "codeberg.org/jhot/hats/pkg/amcrest"

client := amcrest.New(
    cfg.GetCustomSetting("AMCREST_FRONT_DOOR_HOST", ""),
    cfg.GetCustomSetting("AMCREST_FRONT_DOOR_USER", "admin"),
    cfg.GetCustomSetting("AMCREST_FRONT_DOOR_PASS", ""),
)
```

`cfg.GetCustomSetting(name, default)` reads from the environment, so each camera's credentials can be set as individual env vars.

**Config:** custom — use `AMCREST_{NAME}_HOST`, `AMCREST_{NAME}_USER`, `AMCREST_{NAME}_PASS` (or any names you choose)

---


← [Getting started](getting-started.md)
