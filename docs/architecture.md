# Otter Link Architecture

## Overview

Otter Link is a client/server online service inspired by the classic Quantum Link / Q-Link / AOL model.

It is intentionally different from a traditional BBS project. The server provides a shared online service, while clients are designed specifically for the capabilities and conventions of their target platform.

## Core principles

1. **Service first.** The server defines accounts, content, presence, and service APIs independently of any particular client UI.
2. **Platform-native clients.** A C64 client should feel like a C64 application; an Amiga client should feel like an Amiga application; a modern desktop client can use a modern desktop toolkit.
3. **Protocol independence.** Client implementations must not depend on the server's internal database schema.
4. **Clean third-party boundaries.** Open OSCAR remains an independently licensed MIT component/service.
5. **Offline and low-bandwidth awareness.** Retro clients should be able to cache useful state and synchronize efficiently.

## Server layers

```text
+--------------------------------------------------+
|                 Otter Link Server                |
+--------------------------------------------------+
| HTTP API | Client Protocol | OSCAR Compatibility |
+--------------------------------------------------+
| Authentication | Accounts | Sessions | Presence  |
+--------------------------------------------------+
| Forums | Messages | Mail | Files | News | Games  |
+--------------------------------------------------+
| Persistence / Database                           |
+--------------------------------------------------+
```

The current server implementation uses Go and SQLite for the early development system. These choices can be revisited if the project outgrows them.

## Current implementation status

The original server bootstrap milestone is complete. The implementation now includes:

- HTTP health and authentication endpoints
- SQLite account persistence
- Native framed TCP protocol on `:8023`
- OSCAR compatibility listener on `:5190`
- OSCAR authentication and BOS/session handling
- Rate information and location/user-info support
- Buddy list add/delete
- Buddy watcher lookup
- Online/offline presence tracking
- Initial buddy presence delivery
- Presence fan-out to connected watchers

The next implementation focus is end-to-end messaging and broader client interoperability, rather than further expanding the initial server bootstrap.

## Chat / IM

OSCAR compatibility is treated as a separate compatibility layer rather than the definition of the Otter Link service model.

```text
Otter Link Server  <---- service integration ---->  OSCAR compatibility
      GPL                                      compatibility layer
```

The compatibility layer allows existing OSCAR/AIM-style clients to exercise Otter Link accounts, buddy lists, and presence while the native Otter Link protocol continues to evolve.

## Clients

### Modern desktop

Linux, Windows, and macOS will initially target a common Qt 6 / C++ client. The UI may share concepts and code while still allowing platform-specific presentation where useful.

### Amiga / AROS

A native Amiga-style client is the long-term goal. A browser-based client may be useful as an early AROS implementation while the native GUI approach is evaluated against the available AROS APIs and toolchain.

### C64 / C128

The target is a genuinely native retro client, likely using 6502-family assembly and PETSCII-oriented presentation. The C128 may eventually provide enhanced capabilities over the common C64 client.

### Commander X16

The target is a native 65C02/X16 client capable of taking advantage of its additional memory, graphics, and I/O capabilities.

## Client protocol

The public client protocol should expose service operations rather than internal database operations. Initial concepts include:

- Authenticate
- Get/update profile
- List conferences
- List messages
- Retrieve a message and its thread
- Post a message / reply
- Retrieve presence
- Send/receive chat messages
- List/download files
- Retrieve news

The current development protocol is a simple framed, line-oriented JSON transport over TCP. The logical service model is intended to remain independent of this transport so a later compact/binary transport can support constrained retro clients.

## Threaded messages

The message model should support parent/child relationships so conversations form trees. The presentation is intentionally C-Net-inspired: conference listings show the top-level messages, while replies are discovered by entering the original message/thread rather than appearing as a flat list of every reply.

## Milestones

### Complete

1. Start the server.
2. Open/create the SQLite database.
3. Expose a health endpoint.
4. Create and authenticate users.
5. Establish account-backed sessions.
6. Bring up the native client protocol.
7. Bring up the OSCAR compatibility listener.
8. Implement initial buddy list and presence behavior.

### Next

1. Exercise the OSCAR implementation against real clients and capture interoperability gaps.
2. Complete reliable OSCAR messaging/session behavior needed for an end-to-end IM milestone.
3. Expand the native service protocol around the same account, presence, and messaging model.
4. Begin implementing the first actual Otter Link client.

The project should continue to favor small, testable protocol increments over attempting to implement the entire historical service at once.
