# Otter Link

**Your connection to the online world of yesterday.**

Otter Link is an open-source online service inspired by the classic Quantum Link / Q-Link / AOL model. The goal is to provide one shared service that can be reached by modern clients as well as native retro clients for platforms such as Amiga, C64/C128, and Commander X16.

## Current status

Otter Link is in active development. The server foundation is working, and the project has progressed beyond the initial account/session milestone into an early OSCAR-compatible instant-messaging implementation.

Currently implemented or wired into the server:

- Go server with graceful shutdown
- SQLite persistence
- HTTP health endpoint
- Account registration and authentication
- HTTP session/authentication API
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
- Tests covering the server and OSCAR protocol components

This is **not yet a finished AIM replacement or public service**. The OSCAR implementation is being built incrementally, with compatibility and interoperability as the immediate focus.

## Services and ports

| Service | Default | Purpose |
|---|---:|---|
| HTTP API | `:8080` | Health, registration, login, logout, and current-user API |
| Otter Link protocol | `:8023` | Development client/service protocol |
| OSCAR compatibility | `:5190` | AIM/OSCAR-compatible client connectivity |

The HTTP server currently exposes:

- `GET /api/health`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/me`

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

Check the HTTP service with:

```sh
curl http://localhost:8080/api/health
```

Expected response:

```json
{"status":"ok","service":"otter-link"}
```

### Configuration

Environment variables:

- `OTTERLINK_ADDR` — HTTP listen address; default `:8080`
- `OTTERLINK_PROTOCOL_ADDR` — Otter Link protocol listen address; default `:8023`
- `OTTERLINK_OSCAR_ADDR` — OSCAR compatibility listen address; default `:5190`
- `OTTERLINK_DB` — SQLite database path; default `data/otterlink.db`

### Tests

From the `server` directory:

```sh
go test ./...
```

## Repository layout

```text
.
├── docs/       Architecture and protocol documentation
├── protocol/   Reserved for protocol components shared with clients
├── server/     Go server and service implementations
└── tests/      Project-level test material
```

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

## OSCAR compatibility

OSCAR support is being implemented as a compatibility layer so existing AIM/OSCAR clients can be used while Otter Link's native service protocol is developed. The implementation is intentionally incremental: passing a protocol exchange or client startup sequence does not imply that every OSCAR service is supported.

## Project direction

The immediate goal is to turn the current OSCAR foundation into a useful end-to-end messaging experience while keeping the underlying Otter Link service model independent of any one client or legacy protocol.

The long-term goal remains **one service, many clients** — from a modern desktop application to machines that were considered cutting-edge decades ago.
