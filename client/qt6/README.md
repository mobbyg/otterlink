# Otter Link Qt 6 Client

This directory contains the first native desktop client for Otter Link.

The client is intentionally small: it establishes the cross-platform Qt 6 application shell and exercises the existing HTTP service with login, presence, buddies, and community chat.

## Targets

- Linux
- Windows
- macOS

## Requirements

- Qt 6
- CMake 3.21 or newer
- A C++17 compiler

## Build

From this directory:

```sh
cmake -S . -B build
cmake --build build
```

Run `otterlink` from the resulting build directory. The default server is `http://127.0.0.1:9090`; the server field can be changed to point at another Otter Link instance.

## Current development UI

The current client provides:

- Login to an Otter Link server
- A three-stage classic connection presentation: Calling, Connecting, and Connected
- Dashboard with online users and buddies
- Community chat history
- Sending community chat messages
- Manual dashboard refresh
- Automatic dashboard refresh while connected
- Buddy add/remove controls
- Disconnect/logout

The connection presentation is cosmetic: the HTTP login request proceeds normally while the client presents the connection sequence. If the server responds quickly, the client still completes the presentation before entering the dashboard; if authentication fails, the presentation is cancelled and the error is shown.

The current connection artwork is intentionally a temporary text/ASCII otter. It is a placeholder for original Otter Link artwork and keeps this change free of external asset dependencies.

The main window layout is maintained in `mainwindow.ui` and is intended to be edited with Qt Designer/Qt Creator. Keep substantial presentation work in `.ui` files and keep service behavior in C++.

## Presentation direction

The long-term desktop experience is **retro feel, modern engine**. The Qt client will eventually have an Otter Link Classic presentation inspired by the visual language of Q-Link/AOL-era online services while using original Otter Link branding and artwork.

The connection experience is the first implementation of that direction. Future work will replace the placeholder otter with artwork, add optional connection/disconnect audio, and move the presentation into a reusable theme/presentation component.

See `../../docs/presentation.md` for the broader design direction.

## Direction

This is a native client foundation, not a finished UI. Service operations should remain behind the client/service boundary so the same Otter Link account and community can later be presented by native Amiga/AROS, C64/C128, and Commander X16 clients.
