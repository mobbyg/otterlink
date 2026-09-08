# Otter Link Client Protocol

## Status

**Draft — service contract and development transport.**

The logical client protocol is still under development. The current server has a working framed TCP implementation for development and interoperability testing, but the service/action set is not yet frozen and this transport should not be considered the final wire format for constrained retro clients.

## Goals

Otter Link is a service platform, not simply a web API. Clients should be able to use the same account and service model whether they are running on Qt6, AROS/Amiga, C64/C128, Commander X16, or another future client.

The protocol therefore separates:

1. **Transport** — how bytes reach the server.
2. **Session** — authentication and connection state.
3. **Commands/events** — service operations and server notifications.
4. **Service payloads** — chat, presence, mail, boards, and future services.

The current implementation uses a line-oriented, framed JSON protocol over TCP for development and modern-client work. The wire format is intentionally kept simple so a later compact/binary transport can expose the same logical commands to constrained retro clients.

## Current server transports

The server currently exposes three interfaces:

| Transport | Default | Purpose |
|---|---:|---|
| HTTP | `:8080` | Account and web/API operations |
| Otter Link protocol | `:8023` | Native client/service protocol under development |
| OSCAR compatibility | `:5190` | AIM/OSCAR interoperability |

The native protocol and OSCAR compatibility service share the same account/presence backend but are separate wire protocols.

## Transport

### Initial transport

- TCP
- UTF-8
- One message per line
- JSON object per message
- Maximum message size: 64 KiB in the initial implementation

Example:

```json
{"id":1,"type":"request","service":"session","action":"login","payload":{"username":"testotter","password":"..."}}
```

A server response carries the same request ID:

```json
{"id":1,"type":"response","ok":true,"service":"session","action":"login","payload":{"token":"..."}}
```

Asynchronous server events have no request ID:

```json
{"type":"event","service":"presence","action":"changed","payload":{"username":"otter2","status":"online"}}
```

## Message envelope

### Request

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | Client-generated request identifier |
| `type` | yes | `request` |
| `service` | yes | Logical service name |
| `action` | yes | Action within the service |
| `payload` | no | Action-specific object |

### Response

| Field | Required | Meaning |
|---|---|---|
| `id` | yes | Request identifier being answered |
| `type` | yes | `response` |
| `ok` | yes | Whether the request succeeded |
| `service` | yes | Logical service name |
| `action` | yes | Action being answered |
| `payload` | no | Result data |
| `error` | no | Structured error when `ok` is false |

### Event

| Field | Required | Meaning |
|---|---|---|
| `type` | yes | `event` |
| `service` | yes | Event name |
| `action` | yes | Event name |
| `payload` | no | Event-specific data |

## Error format

Errors use stable machine-readable codes rather than requiring clients to parse human text.

```json
{"id":4,"type":"response","ok":false,"service":"session","action":"login","error":{"code":"invalid_credentials","message":"Invalid username or password"}}
```

The `message` is display-oriented and may change. Clients should branch on `code`.

Initial error codes include:

- `bad_request`
- `unauthorized`
- `forbidden`
- `not_found`
- `conflict`
- `rate_limited`
- `server_error`
- `invalid_credentials`
- `session_expired`

## Session model

HTTP API authentication and the client protocol share the same account/session backend but are separate transports.

A client protocol session is intended to progress through:

```text
CONNECT
  ↓
HELLO
  ↓
AUTHENTICATE
  ↓
READY
  ↓
SERVICE REQUESTS / EVENTS
  ↓
DISCONNECT
```

The server must never require a retro client to implement browser-oriented concepts such as cookies, JavaScript, or OAuth.

## Initial services

The service namespace is deliberately explicit so clients can discover and implement only what they support.

### `session`

Initial actions:

- `hello`
- `login`
- `logout`
- `whoami`

### `presence`

Planned actions/events:

- `set_status`
- `get`
- `list`
- `changed`

### `chat`

Planned actions/events:

- `join`
- `leave`
- `say`
- `message`
- `user_joined`
- `user_left`

### Future services

- `mail`
- `boards`
- `files`
- `directory`
- `games`
- `notifications`

Services are independent modules. A client may advertise or discover supported services and degrade gracefully when a service is unavailable.

## Capability negotiation

The first client message will identify the protocol version and client capabilities.

Example:

```json
{"id":1,"type":"request","service":"session","action":"hello","payload":{"protocol":"1","client":"qt","version":"0.1","capabilities":["chat","presence"]}}
```

The server response will identify the protocol version it selected and the services available to that client.

This is important for retro clients. A C64 client should not need to understand services or fields it cannot display.

## Retro-client compatibility

The logical protocol must not assume modern screen sizes, mouse input, Unicode rendering, or large memory.

The protocol design therefore follows these rules:

- Commands have short, stable names.
- Structured data has explicit fields.
- Human-readable text remains available alongside machine-readable values where useful.
- Clients may request reduced payloads.
- Optional fields can be ignored safely.
- A future compact/binary transport can map directly to the same service/action model.

The first TCP/JSON implementation is a development and interoperability protocol, not a commitment that C64 clients will parse JSON forever.

## Security

- Authentication credentials must only be sent over an encrypted transport in production.
- Passwords are never stored in plaintext.
- Session tokens are revocable.
- Clients must treat server-provided text as untrusted data.
- Service authorization is enforced server-side.

TLS support will be added before the protocol is exposed publicly.

## Design principle

**One service, many clients.**

The server owns identity, persistence, permissions, and service state. Clients own presentation and interaction appropriate to their platform.
