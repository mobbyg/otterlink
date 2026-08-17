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
| API / Client Protocol                            |
+--------------------------------------------------+
| Authentication | Accounts | Sessions | Presence  |
+--------------------------------------------------+
| Forums | Messages | Mail | Files | News | Games  |
+--------------------------------------------------+
| Persistence / Database                           |
+--------------------------------------------------+
```

The initial implementation is expected to use Go for the server and SQLite for the early development database. These choices can be revisited if the project outgrows them.

## Chat / IM

Open OSCAR will be treated as a separate service rather than copied into the Otter Link codebase.

```text
Otter Link Server  <---- API / service integration ---->  Open OSCAR
      GPL                                              MIT
```

Otter Link will provide the account and service model, while Open OSCAR can provide AIM/ICQ-compatible instant messaging, presence, and chat capabilities where appropriate.

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

The final wire format should be selected after the service model is established. Modern clients may use HTTPS and WebSockets; retro clients may use a compact protocol through a gateway.

## Threaded messages

The message model should support parent/child relationships so conversations form trees. The presentation is intentionally C-Net-inspired: conference listings show the top-level messages, while replies are discovered by entering the original message/thread rather than appearing as a flat list of every reply.

## Initial milestone

The first implementation milestone is deliberately small:

1. Start the server.
2. Open/create the SQLite database.
3. Expose a health endpoint.
4. Create a user.
5. Authenticate the user.
6. Establish a session.

No client, chat integration, or retro protocol is required for this milestone.
