# Getting started

This guide walks through running HATS and wiring it up to your Home Assistant instance.

Once HATS is running, see the [subscriber guide](subscriber-guide.md) to write your first subscriber.

← [README](../README.md)

---

## Prerequisites

- **Docker** (or a Go toolchain if building from source)
- A running **Home Assistant** instance
- A **long-lived access token** from your HA profile (Profile → Security → Long-lived access tokens → Create token)
- A **NATS server** — the example compose file below includes one

---

## Running HATS

Copy the [`example/compose.yaml`](../example/compose.yaml) to your project and fill in the required variables:

```yaml
environment:
  - HASS_HOST=192.168.1.100    # your Home Assistant IP or hostname
  - HASS_TOKEN=your-ha-token   # long-lived access token
  - NATS_HOST=nats
  - HATS_TOKEN=changeme        # bearer token clients use to call the HATS API
```

Then start it:

```sh
docker compose up -d
```

You can verify HATS is running at `http://localhost:8888` — it renders the README as a homepage. The `/status` endpoint returns `OK` if healthy.

---

## Environment variable reference

Variables are read from the environment in priority order: **Infisical secrets** (if configured) override **environment variables**, which override **defaults**.

### Core

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, `ERROR` |

### Home Assistant

| Variable | Default | Description |
|---|---|---|
| `HASS_HOST` | `127.0.0.1` | Home Assistant hostname or IP address |
| `HASS_PORT` | `8123` | Home Assistant port |
| `HASS_TOKEN` | *(required)* | Long-lived access token from HA profile |
| `HASS_SECURE` | `false` | Set to `true` to use HTTPS and WSS |

### NATS

| Variable | Default | Description |
|---|---|---|
| `NATS_HOST` | `127.0.0.1` | NATS server hostname |
| `NATS_PORT` | `4222` | NATS server port |
| `NATS_TOKEN` | — | Auth token for NATS (optional) |
| `NATS_CLIENT_NAME` | `hats` | Name shown in the NATS monitoring UI |

### HATS API

| Variable | Default | Description |
|---|---|---|
| `HATS_HOST` | `hats` | Hostname used by subscribers to reach the HATS API |
| `HATS_PORT` | `8888` | HATS API port |
| `HATS_TOKEN` | — | Bearer token required by subscribers — leave empty to disable auth |
| `HATS_SECURE` | `false` | Set to `true` for HTTPS |

See [api-reference.md](api-reference.md) for how the token is validated on each request.

### Third-party integrations

These are only needed if you use the corresponding client in your subscriber. See [third-party-clients.md](third-party-clients.md) for details on each.

| Variable | Default | Description |
|---|---|---|
| `NTFY_HOST` | `https://ntfy.sh` | ntfy server URL |
| `NTFY_TOKEN` | — | Auth token for self-hosted ntfy |
| `SYNCTHING_HOST` | `http://127.0.0.1:8384` | Syncthing API URL |
| `SYNCTHING_TOKEN` | — | Syncthing API key |
| `GOKAPI_HOST` | `http://gokapi:53842` | Gokapi URL |
| `GOKAPI_TOKEN` | — | Gokapi API key |
| `QBITTORRENT_HOST` | `http://qbittorrent:8080` | qBittorrent web UI URL |
| `QBITTORRENT_USER` | — | qBittorrent username |
| `QBITTORRENT_PASS` | — | qBittorrent password |

### Infisical secrets

| Variable | Default | Description |
|---|---|---|
| `INFISICAL_HOST` | `http://infisical:8080` | Infisical API URL |
| `INFISICAL_CLIENT` | — | Machine identity client ID |
| `INFISICAL_SECRET` | — | Machine identity client secret |
| `INFISICAL_PROJECT` | — | Project ID |
| `INFISICAL_ENVIRONMENT` | `prod` | Environment slug (`prod`, `dev`, etc.) |

---

## Secret management with Infisical

If `INFISICAL_CLIENT`, `INFISICAL_SECRET`, and `INFISICAL_PROJECT` are all set, HATS fetches secrets from [Infisical](https://infisical.com/) at startup and overlays them on top of environment variables. This lets you keep tokens out of your compose files entirely.

Secret keys must match the environment variable names above (e.g. `HASS_TOKEN`, `HATS_TOKEN`). Custom settings retrieved with `cfg.GetCustomSetting` are also resolved through Infisical if configured.

See [third-party-clients.md#infisical](third-party-clients.md#infisical) for more on the Infisical client.

---

← [README](../README.md) · [Subscriber guide →](subscriber-guide.md)
