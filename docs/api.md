# Otter Link Service API

**Status: development API.** This document describes the HTTP service surface currently implemented by the Go server. It is intended for the Qt 6 client, web client, administration tools, and future clients. The API may change while the service model is developed.

## Base URL

The HTTP listener defaults to:

`http://localhost:9090`

Set `OTTERLINK_ADDR` to change the listen address.

All JSON request bodies must use:

`Content-Type: application/json`

Successful JSON responses use:

`Content-Type: application/json`

## Authentication

Authenticated endpoints use a bearer session token returned by login.

Send:

`Authorization: Bearer <token>`

Sessions are server-side and revocable. Clients should not assume that a token remains valid indefinitely.

Requests without a valid token receive HTTP `401 Unauthorized`.

### Register

`POST /api/auth/register`

Request:

```json
{
  "username": "testotter",
  "display_name": "Test Otter",
  "email": "otter@example.com",
  "password": "example-password"
}
```

Returns HTTP `201 Created` with the created user.

### Login

`POST /api/auth/login`

Request:

```json
{
  "username": "testotter",
  "password": "example-password"
}
```

Returns HTTP `200 OK`:

```json
{
  "user": { "...": "user fields" },
  "token": "session-token"
}
```

### Logout

`POST /api/auth/logout`

Requires authentication. Returns HTTP `204 No Content`.

### Current user

`GET /api/me`

Requires authentication. Returns the authenticated user.

## Presence

### List online users

`GET /api/presence`

Requires authentication.

Returns:

```json
{
  "users": [ "...presence records..." ]
}
```

### Set away status

`POST /api/presence/away`

Requires authentication.

Request:

```json
{
  "away": true
}
```

Returns:

```json
{
  "away": true
}
```

## Buddies

### List buddies

`GET /api/buddies`

Requires authentication.

Returns:

```json
{
  "buddies": [ "...buddy records..." ]
}
```

### Add a buddy

`POST /api/buddies`

Request:

```json
{
  "username": "otter2"
}
```

Returns HTTP `201 Created`.

### Remove a buddy

`DELETE /api/buddies?username=otter2`

Requires authentication. Returns HTTP `204 No Content`.

## Private messages

Private messaging is user-to-user and is separate from Community Chat.

### Get a conversation

`GET /api/messages?with=otter2`

Returns:

```json
{
  "with": { "...": "user record" },
  "messages": [ "...message records..." ],
  "unread_count": 2
}
```

### Send a message

`POST /api/messages`

Request:

```json
{
  "username": "otter2",
  "message": "Hello!"
}
```

Returns HTTP `201 Created` with the created message.

### Mark a conversation read

`POST /api/messages/read`

Request:

```json
{
  "username": "otter2"
}
```

Returns HTTP `204 No Content`.

### Get unread-message counts

`GET /api/messages/unread`

Returns:

```json
{
  "messages": [
    {
      "username": "otter2",
      "count": 3
    }
  ]
}
```

## Community Chat

Community Chat is channel-based. The legacy `/api/chat` routes remain temporarily for compatibility; the channel-aware routes should be used by new clients.

### List channels

`GET /api/chat/channels`

Returns:

```json
{
  "channels": [ "...channel records..." ]
}
```

### Create a temporary channel

`POST /api/chat/channels`

Request:

```json
{
  "name": "Saturday Night Chat",
  "allow_ops_to_create_ops": false
}
```

The authenticated user becomes a member of the newly created temporary channel.

### Get channel state

`GET /api/chat/channels/{channelID}`

Returns the channel, the current user's role, members, and stored messages:

```json
{
  "channel": { "...": "channel record" },
  "role": "user",
  "members": [ "...member records..." ],
  "messages": [ "...message records..." ]
}
```

### Join a channel

`POST /api/chat/channels/{channelID}/join`

Returns channel state including the joining member, current members, and messages.

### Leave a channel

`POST /api/chat/channels/{channelID}/leave`

Returns HTTP `204 No Content`.

### Send a channel message

`POST /api/chat/channels/{channelID}/messages`

Request:

```json
{
  "message": "Hello, everyone!"
}
```

Returns HTTP `201 Created`.

### Change a channel role

`POST /api/chat/channels/{channelID}/users/{username}/role`

Request:

```json
{
  "role": "operator"
}
```

Authorization is enforced by the server according to the channel's role rules.

### Moderate a channel user

`POST /api/chat/channels/{channelID}/users/{username}/moderate`

Request:

```json
{
  "action": "kick"
}
```

Supported actions currently include `kick`, `ban`, and `unban`.

## Service keywords

Service keywords provide AOL/Q-Link-style community shortcuts. Keywords are normalized case-insensitively and returned in uppercase.

### Resolve keyword

`GET /api/keywords/{keyword}`

Requires authentication.

Returns the keyword's display information and associated service targets:

```json
{
  "keyword": "RETRO",
  "display_name": "Retro Computing",
  "description": "Retro community",
  "targets": [
    {"type": "chat", "id": 7, "label": "Retro Computing Chat"},
    {"type": "event", "id": 12, "label": "Retro Computing Night"}
  ]
}
```

Target types currently supported by the registry are `chat`, `event`, and `bulletin`. Bulletin targets are reserved for the Bulletin service integration.

### Administrator keyword management

- `GET /api/admin/keywords`
- `PUT /api/admin/keywords/{keyword}`
- `DELETE /api/admin/keywords/{keyword}`

Admin write requests use:

```json
{
  "display_name": "Retro Computing",
  "description": "Retro community",
  "targets": [
    {"type": "chat", "id": 7, "label": "Retro Computing Chat"},
    {"type": "event", "id": 12, "label": "Retro Computing Night"}
  ]
}
```

The registry is intentionally service-neutral so future services can participate without changing keyword resolution.
## Events

### List events

`GET /api/events?year=2026&month=9`

Returns:

```json
{
  "events": [ "...event records..." ],
  "year": 2026,
  "month": 9
}
```

The list includes events overlapping the requested calendar month.

### Create an event

`POST /api/events`

Request fields:

```json
{
  "title": "Community Meeting",
  "description": "Monthly meeting.",
  "event_type": "public",
  "target_type": "none",
  "target_id": 0,
  "start_at": "2026-09-28T20:00:00-04:00",
  "end_at": "2026-09-28T21:00:00-04:00",
  "all_day": false
}
```

For an all-day event, use `YYYY-MM-DD` values:

```json
{
  "start_at": "2026-09-28",
  "end_at": "2026-09-28",
  "all_day": true
}
```

Event creation and editing permissions are enforced server-side.

### Update an event

`PATCH /api/events/{eventID}`

Uses the same event fields as creation.

### Delete an event

`DELETE /api/events/{eventID}`

Returns HTTP `204 No Content`.

Events are currently limited to dates no more than two years ahead.

## Content screens

Authenticated clients can retrieve the currently active Home screen with GET /api/content/home. The server evaluates publication and start/end scheduling. A published screen with no dates is permanent; clients do not implement scheduling rules themselves.

A screen response contains metadata plus structured JSON content. The current Qt6 renderer understands hero, announcements, services, footer, and an optional background asset. The content format is intentionally structured rather than arbitrary HTML, CSS, or JavaScript. Unknown fields may be ignored by clients.

Individual published or future screens can be retrieved with GET /api/content/screens/{screenID}. Draft/unpublished screens are not exposed through client content endpoints.

### Content administration

The dedicated content editor is available at /admin/content.

Admin endpoints:

- GET /api/admin/content/screens
- PUT /api/admin/content/screens
- DELETE /api/admin/content/screens/{screenID}

The PUT body contains slug, title, published, optional start_at/end_at RFC3339 values, priority, and a structured content JSON object.

Screen version increments when an existing screen is saved. The version is intended to provide a stable cache/version signal as asset delivery and conditional HTTP caching are added.

The current editor is the foundational screen-management slice. The server asset layer provides persistent image storage and cacheable asset delivery, and screen JSON can reference an uploaded asset as its background.

## Content assets

Authenticated clients can retrieve an image asset with:

- GET /api/content/assets/{assetID}

The response includes the stored image bytes and an ETag derived from the asset SHA-256 hash. Assets are cacheable for one year and clients may use If-None-Match to receive HTTP 304 Not Modified when unchanged.

Supported upload types are PNG, JPEG, GIF, and WebP. The maximum upload size is 10 MB.

Admin endpoints:

- GET /api/admin/content/assets
- POST /api/admin/content/assets
- DELETE /api/admin/content/assets/{assetID}

Uploads use multipart/form-data with the file field named asset. The server stores the binary asset outside the database and keeps only metadata in SQLite. The storage directory defaults to data/assets and can be changed with OTTERLINK_ASSET_ROOT.

Asset IDs and hashes are stable references for the content system. A screen can reference an uploaded image as a background with content such as:

```json
{
  "background": {
    "asset": 12,
    "fit": "cover"
  }
}
```

The `asset` value is the asset ID returned by the admin asset endpoint. Supported `fit` values currently include `cover` and `contain`. The Qt6 client and admin Preview both render the referenced background behind the structured screen content.

## Administration API

Administration endpoints require an authenticated account with the `admin` role. The Qt client does not expose these functions.

### Users

- `GET /api/admin/users`
- `GET /api/admin/users/{username}`
- `PATCH /api/admin/users/{username}`
- `POST /api/admin/users/{username}/password`
- `POST /api/admin/users/{username}/sessions/revoke`
- `DELETE /api/admin/users/{username}`

### Audit

- `GET /api/admin/audit`

### Chat administration

- `GET /api/admin/chat/channels`
- `POST /api/admin/chat/channels`
- `DELETE /api/admin/chat/channels/{channelID}`
- `POST /api/admin/chat/channels/{channelID}/users/{username}/role`

### Event administration

- `GET /api/admin/events?year=2026&month=9`
- `POST /api/admin/events`
- `PATCH /api/admin/events/{eventID}`
- `DELETE /api/admin/events/{eventID}`

## HTTP status conventions

Clients should use HTTP status codes for transport-level handling and response bodies for additional information.

Common responses:

| Status | Meaning |
|---|---|
| `200 OK` | Successful request with a response body |
| `201 Created` | Resource successfully created |
| `204 No Content` | Successful request without a response body |
| `400 Bad Request` | Invalid request or service data |
| `401 Unauthorized` | Missing or invalid session |
| `403 Forbidden` | Authenticated but not permitted |
| `404 Not Found` | Requested resource/user does not exist |
| `405 Method Not Allowed` | HTTP method is not supported |
| `410 Gone` | Legacy endpoint no longer performs that operation |
| `415 Unsupported Media Type` | Request is not JSON |
| `500 Internal Server Error` | Unexpected server failure |

Error responses currently use plain text for most HTTP handlers. Clients should not depend on exact human-readable error strings unless a service documents a stable value. The native protocol uses structured machine-readable error codes separately.

## Native client protocol

The native TCP protocol listens on `:8023` by default and is documented separately in [protocol.md](protocol.md).

The HTTP API and native protocol are different transports over the same service model. A client should not depend on server database tables or Go implementation details.

## Service design rules

New services should follow these conventions:

1. Put service behavior in a dedicated server package.
2. Expose service operations through the HTTP API and native protocol as appropriate.
3. Keep authentication and authorization server-side.
4. Return stable structured data rather than UI-specific HTML.
5. Keep presentation decisions in clients.
6. Allow clients to ignore fields they do not understand.
7. Keep service state independent of any one client.
8. Document request fields, response shapes, permissions, and lifecycle rules before treating an API as stable.

The long-term goal is **one service, many clients**: Qt, web, and future native retro clients should consume the same logical services even when their presentation and transport capabilities differ.
