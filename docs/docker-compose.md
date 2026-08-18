# Docker Compose Setup

## Environment Variables

HATS splits configuration into two tiers: **bootstrap** variables that must always be present in the container, and **optional** variables that can be omitted once they have been saved through the admin UI secret store.

### Bootstrap (always required)

These variables must be set in your `docker-compose.yml` because they are needed before the secret store is accessible.

| Variable | Description |
|---|---|
| `NATS_HOST` | NATS server hostname |
| `NATS_PORT` | NATS server port (default `4222`) |
| `NATS_TOKEN` | NATS authentication token |
| `SECRETS_ENCRYPTION_KEY` | 64-char hex string (32 bytes) used to encrypt secrets at rest. Generate with: `openssl rand -hex 32` |
| `ADMIN_PASSWORD_HASH` | bcrypt hash of the admin UI password. Generate with: `htpasswd -bnBC 12 "" yourpassword \| tr -d ':\n'` |
| `ADMIN_SESSION_SECRET` | Random string for signing admin session cookies |

### Optional (can be saved via admin UI instead)

Once you have saved these values through the admin UI at `/admin/homeassistant` or `/admin/secrets`, you can remove them from your compose file. The server reads them from the secret store on startup.

| Variable | Description |
|---|---|
| `HASS_HOST` | Home Assistant hostname |
| `HASS_PORT` | Home Assistant port (default `8123`) |
| `HASS_SECURE` | Set `true` for HTTPS/WSS (default `false`) |
| `HASS_TOKEN` | Home Assistant long-lived access token |
| `NTFY_HOST` | ntfy server URL |
| `NTFY_TOKEN` | ntfy authentication token |
| Any custom secret | Any secret you have created via `/admin/secrets` |

## Restart Policy

The admin UI includes a **Restart Service** button on the Home Assistant connection page. Clicking it gracefully exits the HATS process so Docker can restart it with the freshly-read configuration.

**This requires a Docker restart policy.** Without one the service will exit and not recover.

Add `restart: unless-stopped` (or `restart: always`) to the HATS service in your compose file:

```yaml
services:
  hats:
    image: ghcr.io/jhot21/hats:latest
    restart: unless-stopped        # required for the Restart button to work
    environment:
      NATS_HOST: nats
      NATS_PORT: "4222"
      NATS_TOKEN: your-nats-token
      SECRETS_ENCRYPTION_KEY: "your-64-char-hex-key"
      ADMIN_PASSWORD_HASH: "your-bcrypt-hash"
      ADMIN_SESSION_SECRET: "your-random-string"
      # HASS_HOST, HASS_TOKEN etc. can be omitted once saved via admin UI
    ports:
      - "8888:8888"
    depends_on:
      - nats

  nats:
    image: nats:latest
    command: ["-js", "--auth", "your-nats-token"]
```

## First-Time Setup

1. Set all environment variables (including `HASS_*`) in your compose file and start the stack.
2. Open the admin UI at `http://localhost:8888/admin` and log in.
3. Navigate to **Home Assistant** and verify the connection details are correct.
4. Once the connection test passes, remove the `HASS_*` variables from your compose file.
5. Use the **Restart Service** button to apply the stored configuration.

From this point forward, connection details are managed through the admin UI and do not need to be in the compose file.
