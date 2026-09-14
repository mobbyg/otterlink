<img width="400" height="120" alt="otterlink_logo" src="https://github.com/user-attachments/assets/9da99a3d-298b-4740-9e55-6af0f387c59f" />


**Your connection to the online world of yesterday.**

Otter Link is an open-source online service inspired by the classic Quantum Link / Q-Link / AOL model. The goal is to provide one shared service that can be reached by modern clients as well as native retro clients for platforms such as Amiga, C64/C128, and Commander X16.

## Current status

Otter Link is in active development. The server foundation is working, and the project now has usable web and Qt 6 client surfaces for accounts, buddies, presence, and multi-room community chat.

Currently implemented or wired into the server:

- Go server with graceful shutdown
- SQLite persistence
- HTTP health endpoint
- Account registration and authentication
- HTTP session/authentication API
- Development web client served by the Go server
- Web client dashboard with account, buddy, presence, and community chat views
- Protected web administration foundation with account roles, user management, session management, audit logging, and chat channel management
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
- Multi-room Community Chat with persistent message history
- Temporary user-created chat rooms and permanent administrator-created rooms
- Chat channel roles with original/delegated moderators and Operators
- Chat moderation, bans, kicks, and role management
- Qt 6 Community Chat client with channel switching, room creation, user list, roles, and moderation controls
- Tests covering the server, chat service, and OSCAR protocol components

This is **not yet a finished AIM replacement or public service**. The OSCAR implementation and native service protocol are being built incrementally, with the web and Qt clients serving as practical development surfaces while the service model takes shape.

## Services and ports

| Service | Default | Purpose |
|---|---:|---|
| HTTP / web client | `:9090` | Web UI, health, registration, login, logout, service APIs, and administration |
| Otter Link protocol | `:8023` | Development client/service protocol |
| OSCAR compatibility | `:5190` | AIM/OSCAR-compatible client connectivity |
| OtterWeb | Planned | Integrated web browser, portal, directory, and search service |

Open `http://localhost:9090/` after starting the server to use the development UI.

The initial web administration surface is available at `http://localhost:9090/admin`. Administrative data is protected server-side by the account's `admin` role; the Qt client does not expose administration functions.

The administration surface currently supports account editing, password resets, session revocation, account deletion safeguards, chat channel management, role management, and a recent activity/audit view. Audit entries record administrative actions and outcomes without recording passwords or session tokens.

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
- Instant messaging and community chat
- Mail
- Threaded boards/conferences
- Files
- News and notifications
- Games and other online services
- OtterWeb browsing, directory, and search services

See [`docs/architecture.md`](docs/architecture.md) for the architectural direction and [`docs/protocol.md`](docs/protocol.md) for the developing Otter Link client protocol.

## Development UI

The web client is deliberately a small, dependency-free browser UI rather than a separate frontend application. It is intended to give us a real surface for testing service behavior and to establish the visual and interaction direction for a future modern client.

It currently provides:

- Account registration and login
- Current-user display
- Buddy list management
- Online-user display
- Multi-room community chat
- Automatic refresh while connected

The initial administration surface is intentionally separate from the normal client experience. It is being built as a web/server concern so administrative operations do not become part of the Qt 6 client or the retro client protocol surface.

The Qt 6 client now includes a dedicated Community Chat service window with room selection, room creation, message history, active-user display, channel roles, and moderation controls.

It is a **development client**, not the final Otter Link UI. As the service grows, this surface can evolve into a richer modern client while native clients continue to present the same underlying services in platform-appropriate ways.

## OtterWeb

**OtterWeb is a planned future Otter Link service.**

The goal is to make web browsing feel like a service inside Otter Link rather than simply opening an external browser. The planned experience combines a Qt 6 WebEngine browser, an OtterLink-branded portal/home page, favorites, history, a web directory, a small search index, Modern Web browsing, and Retro Web browsing.

The portal should take inspiration from the late-1990s/early-2000s online-service experience while using original OtterLink/OtterWeb branding and content. Planned directory categories include news, technology, computing, retro computing, games, ham radio, entertainment, community, personal sites, and Otter Link services.

Retro Web support may integrate historical-web providers such as Protoweb, but OtterWeb should treat those providers as integrations rather than making the browser itself dependent on any one provider.

A future OtterWeb index may crawl and index modest amounts of public web content using fields such as URL, title, description, headings, visible text, keywords, category, and last indexed time. Users may eventually be able to submit sites for directory/search inclusion, with moderation before curated placement.

The intended implementation order is:

1. Dedicated Qt WebEngine browser widget
2. Navigation and address/search bar
3. OtterWeb home page
4. Favorites/bookmarks
5. History
6. Modern Web browsing
7. Retro Web/provider support
8. OtterWeb directory
9. Search index
10. Crawler
11. Site submission and moderation

The browser and portal should be useful before the crawler and search system are attempted.

## OSCAR compatibility

OSCAR support is being implemented as a compatibility layer so existing AIM/OSCAR clients can be used while Otter Link's native service protocol is developed. The implementation is intentionally incremental: passing a protocol exchange or client startup sequence does not imply that every OSCAR service is supported.

## Project direction

The immediate goal is to turn the current server, chat, and OSCAR foundation into a useful end-to-end online service, using the web and Qt clients as practical test surfaces while keeping the underlying Otter Link service model independent of any one client or legacy protocol.

The long-term goal remains **one service, many clients** — from a modern desktop application to machines that were considered cutting-edge decades ago.
