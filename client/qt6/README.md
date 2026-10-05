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

Run `otterlink` from the resulting build directory. The default server is `http://127.0.0.1:9090`; the server field can be changed to point at another Otter Link instance. For a TLS-enabled server, enter its `https://` URL (for example, `https://otterlink.example.org:9090`). Qt uses the platform certificate store to validate the server certificate.

## Current development UI

The current client provides:

- Login to an Otter Link server
- A three-stage classic connection presentation: Calling, Connecting, and Connected
- An AOL-inspired desktop workspace after connection
- A persistent service button bar
- Movable internal service windows
- Open/raise behavior that prevents duplicate windows for each service
- Minimize and close controls on service windows
- Home, People, and Community Chat service windows
- Placeholder windows for planned Mail, Boards, News, Files, and Games services
- Dashboard data presented through the existing client/service functions
- Community chat history and message sending
- Manual and automatic dashboard refresh
- Buddy add/remove controls
- Disconnect/logout

The desktop workspace is intentionally a presentation-layer change. Existing service/network behavior is retained rather than redesigned as part of this UI work.

The connection presentation is cosmetic: the HTTP login request proceeds normally while the client presents the connection sequence. If the server responds quickly, the client still completes the presentation before entering the dashboard; if authentication fails, the presentation is cancelled and the error is shown.

The current connection artwork is intentionally a temporary text/ASCII otter. It is a placeholder for original Otter Link artwork and keeps this change free of external asset dependencies.

The main window layout is maintained in `mainwindow.ui` and is intended to be edited with Qt Designer/Qt Creator. Keep substantial presentation work in `.ui` files and keep service behavior in C++.

## Presentation direction

The long-term desktop experience is **retro feel, modern engine**. The Qt client will eventually have an Otter Link Classic presentation inspired by the visual language of Q-Link/AOL-era online services while using original Otter Link branding and artwork.

The connection experience and desktop workspace are the first implementations of that direction. Future work will refine the desktop/window visual language, replace the placeholder otter with original artwork, and consider optional connection/disconnect audio.

See `../../docs/presentation.md` for the broader design direction.

## Direction

This is a native client foundation, not a finished UI. Service operations should remain behind the client/service boundary so the same Otter Link account and community can later be presented by native Amiga/AROS, C64/C128, and Commander X16 clients.

## Windows cross-build

A 64-bit Windows cross-build can be configured from Linux with the repository
toolchain file:

```sh
cmake -S . -B build-windows \
  -DCMAKE_TOOLCHAIN_FILE=cmake/toolchains/mingw-w64-x86_64.cmake \
  -DCMAKE_PREFIX_PATH=/path/to/windows/qt
cmake --build build-windows
```

The Windows Qt installation must be a Qt 6 build for Windows/x86_64 using
MinGW-w64. The Linux host Qt installation is used for the host-side Qt build
tools, while the Windows Qt installation supplies the target headers,
libraries, CMake package files, and Windows deployment tools.

Do not commit Qt SDK binaries or compiler toolchains to the repository.


## Image format support

Server-managed Home graphics can use WebP assets. Qt provides WebP support
through the Qt Image Formats plugin, so the plugin must be available at runtime
on every target.

### Linux

On Ubuntu/Debian systems using the distribution Qt 6 packages, install the
Qt 6 image-format plugins:

```sh
sudo apt install qt6-image-formats-plugins
```

This package supplies the WebP image plugin used by `QImage`/`QImageReader`.

### Windows

When packaging the cross-built Windows client, include the WebP image plugin
from the Windows Qt installation alongside the other Qt plugins:

```text
otterlink-windows/
├── otterlink.exe
├── Qt6Core.dll
├── Qt6Gui.dll
├── Qt6Network.dll
├── Qt6Widgets.dll
├── libgcc_s_seh-1.dll
├── libstdc++-6.dll
├── libwinpthread-1.dll
├── platforms/
│   └── qwindows.dll
└── imageformats/
    └── qwebp.dll
```

For the Qt 6.4.2 MinGW installation used for the Linux cross-build, the source
plugin is:

```text
~/Qt/6.4.2/mingw_64/plugins/imageformats/qwebp.dll
```

The client logs a warning if a Home asset cannot be decoded, including the
Qt image-reader error and detected format, which makes missing image plugins
straightforward to diagnose.


## Transport security

The Qt client uses Qt Network for all HTTP API requests. When the server is
configured with TLS and the client uses an `https://` server URL, the login,
authentication token, Home content, chat, buddy/presence, messages, events, and
other HTTP API traffic are encrypted by TLS in transit.

The client does **not** disable certificate validation. The server certificate
must be trusted by the operating system and must match the hostname used in the
server URL. For local development without a trusted certificate, the existing
`http://127.0.0.1:9090` mode remains available.

The separate native JSON-lines listener (`:8023`) and OSCAR compatibility
listener (`:5190`) are not changed by this HTTP TLS slice. Their transport
security/compatibility requirements are a separate concern.
