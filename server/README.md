# Otter Link Server

Initial Go server foundation.

## Requirements

- Go 1.24 or newer

## Run

From this directory:

```sh
go run .
```

The server listens on `:8080` by default and creates `data/otterlink.db` when started.

Environment variables:

- `OTTERLINK_ADDR` — HTTP listen address; default `:8080`.
- `OTTERLINK_DB` — SQLite database path; default `data/otterlink.db`.

Health check:

```sh
curl http://localhost:8080/api/health
```

Expected response:

```json
{"status":"ok","service":"otter-link"}
```

## Tests

```sh
go test ./...
```
