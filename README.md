<img width="400" height="120" alt="otterlink_logo" src="https://github.com/user-attachments/assets/9da99a3d-298b-4740-9e55-6af0f387c59f" />

**Your connection to the online world of yesterday.**

Otter Link is an open-source online service inspired by the classic Quantum Link / Q-Link / AOL model. It is designed as **one service, many clients**, with modern clients and future native clients for platforms such as Amiga, C64/C128, and Commander X16.

## Current status

Otter Link is an active development project. The server, development web client, Qt 6 client, account system, presence, buddy handling, Community Chat, native protocol foundation, and OSCAR compatibility layer are working pieces of the project.

### Currently implemented

- Go server with graceful shutdown
- SQLite persistence
- HTTP health endpoint
- Account registration and authentication
- HTTP session/authentication API
- Development web client served by the Go server
- Web dashboard with account, buddies, presence, and Community Chat
- Protected web administration
- Account management and session management
- Administrative audit logging
- Native framed TCP client protocol on port `8023`
- OSCAR compatibility service on port `5190`
- OSCAR authentication/login, session/BOS, rate information, and location/user-info flows
- Buddy list add/delete and buddy watcher lookup
- Online/offline presence tracking and presence fan-out
- Multi-room Community Chat with persistent message history
- Temporary user-created rooms and permanent administrator-created rooms
- Chat channel roles, moderation, bans, kicks, unbanning, and role management
- Qt 6 Community Chat service window
- Automated tests for server, chat, and OSCAR components

## What it is not

Otter Link is **not yet a finished AIM replacement or public service**.

The OSCAR implementation is an incremental compatibility layer, not a claim of complete AIM/OSCAR compatibility. The native Otter Link protocol is also still under development.

The web and Qt clients are development clients and practical service test surfaces. They are not yet the final modern client experience, and native retro clients are not yet implemented.

## Services and ports

| Service | Default | Purpose |
|---|---:|---|
| HTTP / web client | `:9090` | Web UI, health, authentication, service APIs, and administration |
| Otter Link protocol | `:8023` | Native development client/service protocol |
| OSCAR compatibility | `:5190` | AIM/OSCAR-compatible client connectivity |

After starting the server, open `http://localhost:9090/` for the development web client.

## HTTP API

Current public HTTP endpoints include:

- `GET /api/health`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/me`
- `GET /api/presence`
- `GET /api/buddies`
- `POST /api/buddies`
- `DELETE /api/buddies?username=...`
- `GET /api/chat/channels`
- `POST /api/chat/channels`
- `GET /api/chat/channels/{id}`
- `POST /api/chat/channels/{id}/join`
- `POST /api/chat/channels/{id}/leave`
- `POST /api/chat/channels/{id}/messages`
- `POST /api/chat/channels/{id}/roles`
- `POST /api/chat/channels/{id}/moderate`
- `POST /api/chat/channels/{id}/unban`
- `/api/admin/chat/*` — admin role required
- `GET /api/admin/users` — admin role required
- `GET /api/admin/audit` — admin role required

Detailed protocol and architecture documentation lives in [`docs/`](docs/).

## Build and run

### Requirements

- Go 1.24 or newer
- A C compiler/toolchain suitable for the Go SQLite driver
- Qt 6 for the native Qt client

### Server

Run directly from the repository:

```sh
cd server
go run .
```

Build a server binary:

```sh
cd server
go build -o otterlink-server .
```

The server creates `data/otterlink.db` by default.

### Tests

From `server`:

```sh
go test ./...
```

## Configuration

Environment variables:

- `OTTERLINK_ADDR` — HTTP listen address; default `:9090`
- `OTTERLINK_PROTOCOL_ADDR` — Otter Link protocol listen address; default `:8023`
- `OTTERLINK_OSCAR_ADDR` — OSCAR compatibility listen address; default `:5190`
- `OTTERLINK_DB` — SQLite database path; default `data/otterlink.db`
- `OTTERLINK_ADMIN_USERNAME` — existing account to grant the `admin` role at server startup; unset by default

For example:

```sh
OTTERLINK_ADMIN_USERNAME=yourusername go run .
```

This promotes the named existing account; it does not create an account or set a password.

## Administration

The web administration surface is available at:

```text
http://localhost:9090/admin
```

Administrative access is protected server-side by the account's `admin` role. The Qt client does not expose administration functions.

Current administration functions include:

- Account editing
- Password resets
- Session revocation
- Account deletion safeguards
- Chat channel management
- Chat role management
- Recent activity/audit view

Audit entries record administrative actions and outcomes without recording passwords or session tokens.

## Administration shell

The repository also includes `server/cmd/otterlink-admin`, a separate command-line administration client for the existing HTTP administration API. It supports interactive administration, one-shot commands, and structured pipelines such as:

```text
users list | where status=active | select username,role
users list | where role=admin | count
chat list
events list
audit list
```

Build it from `server/` with:

```sh
go build -o otterlink-admin ./cmd/otterlink-admin
```

See [`server/cmd/otterlink-admin/README.md`](server/cmd/otterlink-admin/README.md) for the command reference and authentication details.

## Development clients

### Web client

The web client is embedded in the Go server and requires no separate frontend build system. It currently provides:

- Registration and login
- Current-user information
- Buddy list management
- Online-user display
- Multi-room Community Chat
- Automatic refresh while connected

### Qt 6 client

The Qt client currently provides the modern desktop development surface, including the Community Chat service window with room selection, room creation, message history, active-user display, roles, and moderation controls.

The desktop presentation is still under development. The client architecture keeps service-specific behavior in dedicated service components rather than putting it into generic window infrastructure.

## Repository layout

```text
.
├── docs/       Architecture and protocol documentation
├── protocol/  Reserved for protocol components shared with clients
├── server/    Go server and service implementations
└── tests/     Project-level test material
```

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — service and client architecture
- [`docs/protocol.md`](docs/protocol.md) — developing Otter Link protocol

Future services and client features are tracked in GitHub issues rather than maintained as a roadmap in this README. The root README is intentionally kept focused on **what exists, what does not yet exist, and how to build, run, test, and administer the current system**.
