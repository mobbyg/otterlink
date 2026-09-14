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
6. **Retro presentation, modern engine.** The service infrastructure remains modern while clients may deliberately evoke the connected-computer experience of the 1980s and 1990s.
7. **Specific behavior, specific ownership.** Service-specific behavior should live in service-specific components rather than being generalized into shared UI infrastructure without a clear need.

## Server layers

```text
+--------------------------------------------------+
|                 Otter Link Server                |
+--------------------------------------------------+
| HTTP API | Client Protocol | OSCAR Compatibility |
+--------------------------------------------------+
| Authentication | Accounts | Sessions | Presence  |
+--------------------------------------------------+
| Chat | Forums | Messages | Mail | Files | News   |
| Games | OtterWeb and other online services       |
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
- Multi-room persistent Community Chat
- Temporary user-created chat rooms and permanent administrator-created rooms
- Original and delegated moderator roles plus Operator roles
- Chat moderation, bans, kicks, unbanning, and role management
- Protected web administration for chat channels and roles
- A dedicated Qt 6 Community Chat service window

The next implementation focus is building out the service model and client experience around these working foundations, rather than expanding the initial server bootstrap in isolation.

## Chat / IM

Community Chat is implemented as a dedicated server service with persistent SQLite-backed channel state and message history. It is exposed through the web API and the Qt client rather than being tied to either presentation.

Channels may be temporary user-created rooms or permanent administrator-created rooms. Temporary rooms are removed when all users leave. Permanent channels are managed through the protected administration surface.

Each channel has a role hierarchy:

- **User** — ordinary channel member
- **Operator** — may moderate ordinary users and may create Operators when the channel permits it
- **Moderator** — may moderate users and Operators and may assign Operator roles
- **Original Moderator** — the channel's original creator/moderator, with additional ownership semantics

The original creator automatically regains Original Moderator status when returning to the channel. Delegated moderators do not receive that automatic restoration. Only the Original Moderator can create another Moderator.

Moderation and role assignment are enforced by the server, not merely by the client UI. The Qt client presents the appropriate controls for the current user's role.

The service currently exposes channel-oriented operations including:

- List channels
- Create a channel
- Retrieve channel state
- Join/leave a channel
- Send and retrieve messages
- Assign channel roles
- Kick or ban users
- Unban users

## OSCAR compatibility

OSCAR compatibility is treated as a separate compatibility layer rather than the definition of the Otter Link service model.

```text
Otter Link Server  <---- service integration ---->  OSCAR compatibility
      GPL                                      compatibility layer
```

The compatibility layer allows existing OSCAR/AIM-style clients to exercise Otter Link accounts, buddy lists, and presence while the native Otter Link protocol continues to evolve.

## Clients

### Modern desktop

Linux, Windows, and macOS will initially target a common Qt 6 / C++ client. The UI may share concepts and code while still allowing platform-specific presentation where useful.

The Qt client should use Qt Designer `.ui` files for substantial layouts rather than growing a monolithic hand-built window. Service behavior and client state remain in C++ behind that presentation layer.

The intended desktop experience is **retro feel, modern engine**. The default classic presentation can draw from the visual language of Q-Link/AOL-era online services while using original Otter Link branding and artwork. The client should retain sensible modern behavior rather than reproducing historical limitations.

The client should eventually provide an optional dial-up-style connection presentation with three conceptual stages — Calling, Connecting/Carrier, and Connected — synchronized with the actual connection state. Connection audio should be independently optional from the visual sequence, with separate controls for connection and disconnect effects.

Service windows should own service-specific behavior. For example, Community Chat uses a dedicated `OtterChatWidget` rather than adding chat-specific state and controls to the generic service-window implementation.

See [`docs/presentation.md`](presentation.md) for the presentation and connection-experience direction.

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

## OtterWeb

OtterWeb is the planned web-browsing, directory, and search service for Otter Link.

The goal is to make web browsing feel like a service inside Otter Link rather than simply opening an external browser. The planned modern implementation uses a dedicated Qt 6 `OtterBrowserWidget` backed by Qt WebEngine.

```text
Otter Link Qt Client
        |
        +-- OtterWebWidget
        |      |
        |      +-- Qt WebEngine
        |      +-- Favorites / History
        |      +-- OtterWeb Home / Portal
        |      +-- OtterWeb Directory
        |      +-- Search
        |
        +-- Otter Link services
```

Planned browser capabilities include:

- Back / forward navigation
- Reload
- Home
- Address/search field
- Favorites/bookmarks
- History
- Modern Web browsing
- Retro Web browsing

The default home page should be an OtterLink-branded portal inspired by the late-1990s/early-2000s online-service experience. It should provide search, directory categories, featured sites, community links, Otter Link service links, and an entry point to Retro Web content while using original branding and content.

Retro Web support may integrate historical-web providers such as Protoweb. Providers should remain integrations rather than becoming hard dependencies of the browser architecture.

The planned OtterWeb directory may initially be curated manually, with categories such as News, Technology, Computing, Retro Computing, Games, Ham Radio, Entertainment, Community, Personal Sites, and Otter Link Services.

A modest future search index can crawl public web content and index fields such as URL, page title, description, headings, visible text, keywords, category, and last indexed time. Otter Link's own services can also be indexed. Users may eventually submit sites for directory/search inclusion, with moderation before curated placement.

OtterWeb is deliberately not intended to become a replacement for Google. The useful milestone is an integrated portal, browser, directory, and small search service that fits the Otter Link experience.

Implementation order:

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
9. Establish the first usable web development client.
10. Establish the first usable Qt 6 native desktop client foundation.
11. Implement persistent multi-room Community Chat and its moderation model.
12. Add protected web administration for chat channels and roles.
13. Add the Qt 6 Community Chat service window.

### Next

1. Exercise the OSCAR implementation against real clients and capture interoperability gaps.
2. Complete reliable OSCAR messaging/session behavior needed for an end-to-end IM milestone.
3. Expand the native service protocol around the same account, presence, chat, and messaging model.
4. Continue building the Qt client into a practical development/test client with messaging, buddy management, chat, and live refresh.
5. Begin the client-side presentation/theme layer, keeping the retro experience independent of the service implementation.
6. Add the optional dial-up-style connection presentation and independently controlled connection audio.
7. Begin the OtterWeb browser and portal milestone with a dedicated browser widget.
8. Begin implementing the first actual retro-platform client.

The project should continue to favor small, testable service increments over attempting to implement the entire historical service at once.
