# Otter Link

**Your connection to the online world of yesterday.**

Otter Link is an open-source online service inspired by the classic Quantum Link / Q-Link / AOL model. The goal is to provide one shared service that can be reached by modern clients as well as native retro clients for platforms such as Amiga, C64/C128, and Commander X16.

## Current status

Otter Link is in active development. The server foundation is working, and the project now has first usable web and Qt 6 client surfaces for exercising the account, buddy, presence, and chat portions of the service.

Currently implemented or wired into the server:

- Go server with graceful shutdown
- SQLite persistence
- HTTP health endpoint
- Account registration and authentication
- HTTP session/authentication API
- Development web client served by the Go server
- Web client dashboard with account, buddy, presence, and community chat views
- Initial protected web administration foundation with account roles, user management, session management, and an admin activity log
- Native framed TCP client protocol on port `8023`
- OSCAR compatibility service on port `5190`
- OSCAR authentication/login flow
- OSCAR session/BOS handling
- OSCAR rate information handling
- OSCAR location/user-info queries, including Query2
- Buddy list add/delete
- Buddy watcher lookup
- Online/offline presence tracking
- Initial buddy presence delivery
- Presence fan-out to connected watchers
- Shared in-memory chat history used by the web client and native protocol
- Tests covering the server, chat service, and OSCAR protocol components

This is **not yet a finished AIM replacement or public service**. The OSCAR implementation and native service protocol are being built incrementally, with the web client serving as a practical development surface while the service model takes shape.

## Services and ports

| Service | Default | Purpose |
|---|---:|---|
| HTTP / web client | `:9090` | Web UI, health, registration, login, logout, service APIs, and administration |
| Otter Link protocol | `:8023` | Development client/service protocol |
| OSCAR compatibility | `:5190` | AIM/OSCAR-compatible client connectivity |

Open `http://localhost:9090/` after starting the server to use the development UI.

The initial web administration surface is available at `http://localhost:9090/admin`. Administrative data is protected server-side by the account's `admin` role; the Qt client does not expose administration functions.

The administration surface currently supports account editing, password resets, session revocation, account deletion safeguards, and a recent activity/audit view. Audit entries record administrative actions and outcomes without recording passwords or session tokens.

The HTTP API also exposes:

- `GET /api/health`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/me`
- `GET /api/presence`
- `GET /api/buddies`
- `POST /api/buddies`
- `DELETE /api/buddies?username=...`
- `GET /api/chat`
- `POST /api/chat`
- `GET /api/admin/users` — admin role required
- `GET /api/admin/audit` — admin role required

## Quick start

### Requirements

- Go 1.24 or newer
- SQLite support is provided by the Go SQLite driver; no separate SQLite server is required

### Run the server

```sh
cd server
go run .
```

By default this starts all three server interfaces and creates `data/otterlink.db`.

Then open `http://localhost:9090/` in a browser.

Check the HTTP service directly with:

```sh
curl http://localhost:9090/api/health
```

Expected response:

```json
{"status":"ok","service":"otter-link"}
```

### Configuration

Environment variables:

- `OTTERLINK_ADDR` — HTTP listen address; default `:9090`
- `OTTERLINK_PROTOCOL_ADDR` — Otter Link protocol listen address; default `:8023`
- `OTTERLINK_OSCAR_ADDR` — OSCAR compatibility listen address; default `:5190`
- `OTTERLINK_DB` — SQLite database path; default `data/otterlink.db`
- `OTTERLINK_ADMIN_USERNAME` — existing account to grant the `admin` role at server startup; unset by default

For example, after creating your normal account:

```sh
OTTERLINK_ADMIN_USERNAME=yourusername go run .
```

The setting only promotes the named existing account; it does not create an account or set a password.

### Tests

From the `server` directory:

```sh
go test ./...
```

## Repository layout

```text
.
├── docs/       Architecture and protocol documentation
├── protocol/  Reserved for protocol components shared with clients
├── server/    Go server and service implementations
└── tests/     Project-level test material
```

The development web client lives under `server/internal/web` and is embedded into the Go server, so it does not require a separate frontend build system.

## Architecture

The server owns identity, persistence, permissions, sessions, and service state. Clients are responsible for presentation and interaction appropriate to their platform.

The long-term service model includes:

- Accounts and profiles
- Presence
- Instant messaging and chat
- Mail
- Threaded boards/conferences
- Files
- News and notifications
- Games and other online services

See [`docs/architecture.md`](docs/architecture.md) for the architectural direction and [`docs/protocol.md`](docs/protocol.md) for the developing Otter Link client protocol.

## Development UI

The web client is deliberately a small, dependency-free browser UI rather than a separate frontend application. It is intended to give us a real surface for testing service behavior and to establish the visual and interaction direction for a future modern client.

It currently provides:

- Account registration and login
- Current-user display
- Buddy list management
- Online-user display
- Shared community chat
- Automatic refresh while connected

The initial administration surface is intentionally separate from the normal client experience. It is being built as a web/server concern so administrative operations do not become part of the Qt 6 client or the retro client protocol surface.

It is a **development client**, not the final Otter Link UI. As the service grows, this surface can evolve into a richer modern client while native clients continue to present the same underlying services in platform-appropriate ways.

## OSCAR compatibility

OSCAR support is being implemented as a compatibility layer so existing AIM/OSCAR clients can be used while Otter Link's native service protocol is developed. The implementation is intentionally incremental: passing a protocol exchange or client startup sequence does not imply that every OSCAR service is supported.

## Project direction

The immediate goal is to turn the current server and OSCAR foundation into a useful end-to-end online service, using the development web client as a practical test surface while keeping the underlying Otter Link service model independent of any one client or legacy protocol.

The long-term goal remains **one service, many clients** — from a modern desktop application to machines that were considered cutting-edge decades ago.
