This file provides guidance to AI agents when working with code in this repository.

## For Superpowers Plugin

Keep design, implementation, and other generated documents in `docs/superpowers`

## Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./internal/util/... -run TestFunctionName -v

# Run with race detector
go test -race ./...

# Build the Docker image (multi-platform, macOS only via Taskfile)
task docker-build VERSION=x.x.x PUSH=false

# Tag and push a release
task tag VERSION=x.x.x
```

The Dockerfile itself runs tests in a builder stage (`go test -v ./...`), so a successful image build confirms tests pass.

## Codeberg Workflow (fj CLI)

This repo is hosted on Codeberg (`codeberg.org/jhot/hats`). Use `fj` (Forgejo CLI) instead of `gh` for PRs, issues, and repo operations.

```bash
# Create a PR (autofill title/body from commits)
fj pr create --base main --head <branch> --autofill

# Create a PR with explicit title
fj pr create --base main --head <branch> "feat: my feature"

# View a PR
fj pr view <number>

# List PRs
fj pr search

# Merge a PR
fj pr merge <number>
```

Note: `fj` does not use `-r` or `--repo` flags for repo targeting in subcommands. It infers the repo from the git remote or current directory context.

## Architecture

**HATS** is a bridge that streams Home Assistant events to NATS JetStream and provides a cached HTTP API for state.

```
Home Assistant ──ws──> HATS ──pub──> NATS subjects
                          │
                          └──> HTTP API (state cached in JetStream KV, 2hr TTL)
```

### Package layout

- **`internal/`** — application-only code (not importable by consumers)
  - `internal/homeassistant/` — WebSocket subscriber that connects to HA and republishes events to NATS; handles reconnection with exponential backoff (10 attempts, 5–50s) and ping/pong keepalive
  - `internal/nats/` — NATS/JetStream client wrapper, server-side timers (`gocron`), and cron schedules
  - `internal/api/` — Chi HTTP router: state endpoints, timer/schedule management, ntfy proxy, command routing (HTTP→NATS)
  - `internal/util/` — small helpers (time, OS, Resty); only place with unit tests currently

- **`pkg/`** — reusable public libraries (intended for downstream subscriber services)
  - `pkg/config/` — config struct binding env vars via reflection/tags
  - `pkg/client/` — `HatsClient`: high-level HTTP client for the HATS API (state, timers, schedules, ntfy)
  - `pkg/homeassistant/` — HA REST client, data structs (domain/service constants, `StateData`, `AttributesData`), history API, helper functions
  - `pkg/nats/` — `NatsConnection` builder-pattern wrapper; `AddStateSub()`, `AddEventSub()`, `AddRawSub()` subscriber helpers
  - `pkg/ntfy/` — ntfy message structs
  - `pkg/passgen/` — password generator
  - `pkg/resty/` — custom Resty extensions

- **`pkg/` third-party integrations** — thin clients for external services: `amcrest`, `cocorahs`, `gokapi`, `purpleair`, `syncthing`, `plugins`

- **`cmd/healthcheck/`** — tiny binary used as Docker `HEALTHCHECK`

### NATS subject conventions

Events published by the HA subscriber:
- `homeassistant.states.{domain}.{entity}.{state}`
- `homeassistant.attributes.{domain}.{entity}.{state}`
- `homeassistant.zha.{device_ieee}`
- `homeassistant.zwave-scene.{device_id}`
- `homeassistant.nfc.{tag_id}`

### Configuration

All configuration comes from environment variables. Key vars:
- `HASS_HOST`, `HASS_PORT`, `HASS_TOKEN`, `HASS_SECURE` — Home Assistant WebSocket
- `NATS_HOST`, `NATS_PORT`, `NATS_TOKEN`, `NATS_CLIENT_NAME` — NATS broker
- `HATS_HOST`, `HATS_PORT`, `HATS_TOKEN`, `HATS_SECURE` — HATS HTTP API (for consumer services)
- `NTFY_HOST`, `NTFY_TOKEN` — push notifications
- `LOG_LEVEL` — DEBUG, INFO, WARN, ERROR
- `CONFIG_DIR` — default `/config`

See `pkg/config/config.go` for the full struct with all tags.

### Key patterns

- **Builder/options pattern** — `NatsConnection` accepts functional option args (`WithHostName()`, `WithToken()`, etc.)
- **Context-based cancellation** — `main.go` wires a `context.Context` through everything for graceful shutdown
- **JetStream KV caching** — entity state is stored in a KV bucket with a 2-hour TTL; the HTTP API reads from this cache rather than polling HA directly
- **Concurrent service calls** — `pkg/homeassistant/rest.go` `CallServices()` uses `sync.WaitGroup` for parallel HA service invocations
- **Reflection-based config** — `pkg/config` uses struct tags to bind env vars without manual mapping

### Module path

`codeberg.org/jhot/hats` — repo is hosted on Codeberg, not GitHub.
