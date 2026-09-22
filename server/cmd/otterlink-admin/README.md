# Otter Link Administration Shell

\`otterlink-admin\` is the command-line administration client for the Otter Link server.

It is a separate executable inside the server Go module. It talks to the existing HTTP administration API rather than opening SQLite directly. Authentication, authorization, audit logging, and server-side safety rules therefore remain authoritative in the server.

## Build

From \`server/\`:

\`\`\`sh
go build -o otterlink-admin ./cmd/otterlink-admin
\`\`\`

Or run it directly:

\`\`\`sh
go run ./cmd/otterlink-admin
\`\`\`

## Authentication

Prefer an existing session token:

\`\`\`sh
export OTTERLINK_TOKEN='your-session-token'
./otterlink-admin
\`\`\`

The token can also be supplied with \`--token\`.

With no token, an interactive shell logs in using a username and password:

\`\`\`text
./otterlink-admin
Username: yourusername
Password:
\`\`\`

For non-interactive login, \`OTTERLINK_USERNAME\` and \`OTTERLINK_PASSWORD\` are supported, although a session token is preferred.

The default server is \`http://localhost:9090\`. Override it with \`--server\` or \`OTTERLINK_SERVER\`.

## Users

\`\`\`text
users list
users get USER
users update USER --display-name "New Name" --email user@example.com --status active --role user
users password USER
users revoke USER
users delete USER
\`\`\`

Password changes and account deletion prompt for the required password rather than putting it in command history.

## Chat

\`\`\`text
chat list
chat create "My Community"
chat create "Ops Room" --allow-ops
chat delete CHANNEL_ID
chat role CHANNEL_ID USER mod
chat role CHANNEL_ID USER op
chat role CHANNEL_ID USER user
\`\`\`

Role management is still enforced by the server and applies to permanent channels.

## Events

Server events use the existing admin event API:

\`\`\`text
events list
events list --year 2026 --month 9
events create --title "Town Hall" --description "Monthly meeting" --start-at "2026-10-01T19:00:00Z" --end-at "2026-10-01T20:00:00Z" --all-day false
events update EVENT_ID --title "Updated Town Hall" --description "Updated" --start-at "2026-10-01T19:00:00Z" --end-at "2026-10-01T20:00:00Z"
events delete EVENT_ID
\`\`\`

## Audit and presence

\`\`\`text
audit list
presence list
\`\`\`

## Pipelines

The shell pipes structured records instead of formatted terminal text. Pipeline commands operate on fields from the server response.

\`\`\`text
users list | where status=active
users list | where role=admin | select username,role
users list | sort username
users list | sort -username
users list | head 10
users list | tail 10
users list | count
users list | where status=disabled | select username
\`\`\`

Built-in transforms:

- \`where field=value\`
- \`select field,field,...\`
- \`sort field\`
- \`sort -field\`
- \`count\`
- \`head [N]\`
- \`tail [N]\`

Final output can be JSON for Unix tooling:

\`\`\`sh
./otterlink-admin --json users list
./otterlink-admin --json "users list | where role=admin | select username"
\`\`\`

## Design

The first version intentionally uses only the Go standard library and the HTTP API already exposed by Otter Link.

The shell is not a second authorization system. Existing server safeguards remain in force, including protection against disabling or demoting the administrator's own account and deletion of the last active administrator.

The shell is intentionally small so additional commands, tab completion, richer pipeline transforms, and future Otter Link services can be added without changing the client/server boundary.
