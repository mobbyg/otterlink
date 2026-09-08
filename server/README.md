# Otter Link Server

The Otter Link server is the Go service implementation. It currently provides the HTTP API, the native development protocol, and an OSCAR compatibility listener backed by a shared SQLite account database.

## Requirements

- Go 1.24 or newer

## Run

From this directory:

```sh
go run .
```

The server starts three listeners by default:

- HTTP API: `:8080`
- Otter Link protocol: `:8023`
- OSCAR compatibility: `:5190`

It also creates `data/otterlink.db` when started.

## HTTP API

Health check:

```sh
curl http://localhost:8080/api/health
```

Expected response:

```json
{"status":"ok","service":"otter-link"}
```

Current endpoints:

- `GET /api/health`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/me`

## Configuration

Environment variables:

- `OTTERLINK_ADDR` — HTTP listen address; default `:8080`.
- `OTTERLINK_PROTOCOL_ADDR` — native client protocol listen address; default `:8023`.
- `OTTERLINK_OSCAR_ADDR` — OSCAR compatibility listen address; default `:5190`.
- `OTTERLINK_DB` — SQLite database path; default `data/otterlink.db`.

## OSCAR compatibility

The OSCAR listener is an incremental compatibility implementation. Current server-side support includes authentication, BOS/session handling, rate information, location/user-info requests, buddy list changes, buddy watching, and presence tracking/fan-out.

The compatibility layer is not yet a complete AIM/OSCAR service. Unsupported SNACs may simply receive no response while the implementation is expanded.

## Tests

Run the complete Go test suite with:

```sh
go test ./...
```
