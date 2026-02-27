# API reference

HATS exposes an HTTP API on port `8888` (configurable via `HATS_PORT`). The homepage at `/` renders the README as HTML.

Most subscriber code uses [HatsClient](hats-client-vs-ha-client.md) rather than calling these endpoints directly — HatsClient is a typed Go wrapper around this API. This reference is useful if you're calling HATS from a non-Go service, a Home Assistant automation, or a shell script.

---

## Authentication

All `/api/*` routes require a token when `HATS_TOKEN` is set. Two schemes are accepted:

**Bearer token (recommended):**
```
Authorization: Bearer your-hats-token
```

**HTTP Basic Auth:**
```
Authorization: Basic <base64(any-username:your-hats-token)>
```
The username is ignored; only the password is checked against `HATS_TOKEN`.

If `HATS_TOKEN` is empty, all requests are allowed without authentication.

---

## Endpoints

### Public (no auth)

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | README rendered as HTML |
| `GET` | `/status` | Returns `OK` — health check |
| `GET` | `/passgen` | Generate a deterministic password |

#### `GET /passgen`

Query parameters:

| Parameter | Default | Description |
|---|---|---|
| `salt` | — | Salt string |
| `passphrase` | — | Passphrase to derive the password from |
| `length` | `40` | Output length in characters |
| `version` | `2` | Algorithm version (`1` or `2`) |
| `specials` | — | Override the set of special characters |

---

### Home Assistant state proxy

Responses are served from the JetStream KV cache when available, with a fallback to a live HA API call. See [hats-client-vs-ha-client.md](hats-client-vs-ha-client.md) for caching behavior.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/state/{entityId}` | token | Get entity state as plain text |
| `GET` | `/api/state/{entityId}?full=true` | token | Get full `StateData` as JSON (includes attributes) |
| `POST` | `/api/state/{entityId}/{service}` | token | Call a Home Assistant service on an entity |

#### `GET /api/state/{entityId}`

Returns the state string (e.g. `on`, `off`, `21.5`). Served from the KV cache if available; falls back to the HA REST API and refreshes the cache.

```sh
curl http://hats:8888/api/state/binary_sensor.front_door \
  -H "Authorization: Bearer changeme"
# → on
```

#### `POST /api/state/{entityId}/{service}`

Proxies a service call to Home Assistant. The entity's domain is derived from the entity ID unless overridden.

```sh
# Turn off a light
curl -X POST http://hats:8888/api/state/light.living_room/turn_off \
  -H "Authorization: Bearer changeme"

# With service data
curl -X POST http://hats:8888/api/state/light.living_room/turn_on \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"brightness_pct": 50}'

# Override domain (e.g. automation, scene)
curl -X POST "http://hats:8888/api/state/scene.movie_night/turn_on?domain=scene" \
  -H "Authorization: Bearer changeme"
```

---

### Timers

See [timers-and-schedules.md](timers-and-schedules.md) for the Go client API and usage patterns.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/timer/{name}` | token | Get timer status |
| `POST` | `/api/timer/{name}` | token | Start a timer |
| `DELETE` | `/api/timer/{name}` | token | Cancel a timer |

#### `POST /api/timer/{name}`

Request body (JSON):

| Field | Type | Default | Description |
|---|---|---|---|
| `duration` | string | previous duration | Go duration string: `"5m"`, `"30s"`, `"1h30m"` |
| `force` | bool | `false` | If `true`, restarts the timer even if already running |

```sh
curl -X POST http://hats:8888/api/timer/motion_off \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"duration":"10m","force":false}'
```

Returns the timer status string.

---

### Schedules

See [timers-and-schedules.md](timers-and-schedules.md) for the Go client API and usage patterns.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/schedule/{name}` | token | Get next scheduled fire time |
| `POST` | `/api/schedule/{name}` | token | Create or update a schedule |
| `DELETE` | `/api/schedule/{name}` | token | Delete a schedule |

#### `POST /api/schedule/{name}`

Request body (JSON):

| Field | Type | Required | Description |
|---|---|---|---|
| `cron` | string | yes | 6-field cron expression (seconds first): `"0 0 * * * *"` |

```sh
curl -X POST http://hats:8888/api/schedule/hourly \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"cron":"0 0 * * * *"}'
```

Returns the next scheduled fire time.

---

### Notifications

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/ntfy` | token | Send a notification via ntfy |

#### `POST /api/ntfy`

Proxies a notification to the configured ntfy server. Request body is a JSON `ntfy.Message`:

```sh
curl -X POST http://hats:8888/api/ntfy \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"topic":"alerts","title":"Motion","message":"Front door at 10:32 PM","priority":4}'
```

---

### Commands

See [commands.md](commands.md) for the subscriber-side setup.

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/command/{name}` | token | Publish POST body to `command.{name}` NATS subject |

The POST body can be any content — it is forwarded to NATS verbatim as raw bytes.

```sh
curl -X POST http://hats:8888/api/command/bedtime \
  -H "Authorization: Bearer changeme"

curl -X POST http://hats:8888/api/command/sonarr \
  -H "Authorization: Bearer changeme" \
  -H "Content-Type: application/json" \
  -d '{"seriesTitle":"Fallout"}'
```

---

← [Commands](commands.md)
