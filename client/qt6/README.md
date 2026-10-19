# Otter Link Qt 6 Client

This directory contains the native desktop client for Otter Link.

## Targets

- Linux
- Windows
- macOS

## Requirements

- Qt 6 Core, Multimedia, Network, and Widgets
- CMake 3.21 or newer
- A C++17 compiler
- Qt WebEngine Widgets is recommended on Linux and macOS for the full embedded OtterWeb browser

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
- Server-managed Home screens, including graphics and clickable destinations
- People service with buddy/presence information and private messaging
- Community Chat service window with room selection, message history, active-user display, roles, and moderation controls
- News service window with RSS/Atom articles and native OtterLink-published articles
- Article-reader window for supported news content
- OtterWeb toolbar with address field, Back/Forward/Home, and saved bookmarks
- Bundled Otter Link Search page with a disabled `COMING SOON` field
- Embedded Chromium-based browsing when Qt WebEngine Widgets is available
- A `QTextBrowser` fallback for builds where WebEngine is unavailable, including the current Windows MinGW cross-build
- Placeholder windows for planned Mail, Boards, Files, and Games services
- Dashboard data presented through the existing client/service functions
- Community chat history and message sending
- Manual and automatic dashboard refresh
- Buddy add/remove controls
- Disconnect/logout

The desktop workspace is intentionally a presentation-layer feature, but the Home and News windows now consume server-managed services rather than acting as placeholders. Home screens and native news articles can be published through the web administration interface without rebuilding the client. Existing service/network behavior remains behind the client/service boundary.

The connection presentation is cosmetic: the HTTP login request proceeds normally while the client presents the connection sequence. If the server responds quickly, the client still completes the presentation before entering the dashboard; if authentication fails, the presentation is cancelled and the error is shown.

The main window layout is maintained in `mainwindow.ui` and is intended to be edited with Qt Designer/Qt Creator. OtterWeb uses `QWebEngineView` when the Qt WebEngine Widgets module is installed, allowing modern HTML, CSS, JavaScript, and external websites to render inside the client. If that module is missing, the client falls back to Qt Widgets' `QTextBrowser`; in that fallback mode external HTTP/HTTPS sites open in the system browser.

## Linux: install Qt WebEngine

On Ubuntu 24.04, install the Qt WebEngine development package and its runtime dependencies with:

```sh
sudo apt update
sudo apt install qt6-webengine-dev
```

The project detects `Qt6::WebEngineWidgets` automatically at configure time. After installing the package, rerun CMake configuration so the component is detected:

```sh
cmake -S . -B build
cmake --build build
```

CMake prints whether OtterWeb is using Qt WebEngine or the `QTextBrowser` fallback. Ubuntu 24.04 provides Qt WebEngine 6.4.2 in its repositories. See the [Ubuntu package listing](https://packages.ubuntu.com/noble/qt6-webengine-dev).

## Windows cross-build

A 64-bit Windows cross-build can be configured from Linux with the repository toolchain file:

```sh
cmake -S . -B build-windows \
  -DCMAKE_TOOLCHAIN_FILE=cmake/toolchains/mingw-w64-x86_64.cmake \
  -DCMAKE_PREFIX_PATH=/path/to/windows/qt
cmake --build build-windows
```

The Windows Qt installation must be a Qt 6 build for Windows/x86_64 using MinGW-w64. The Linux host Qt installation is used for the host-side Qt build tools, while the Windows Qt installation supplies the target headers, libraries, CMake package files, and Windows deployment tools. WebEngine is deliberately not required for this Windows cross-build, so the Windows build remains compatible with the existing MinGW kit.

Do not commit Qt SDK binaries or compiler toolchains to the repository.

## Image format support

Server-managed Home graphics can use WebP assets. Qt provides WebP support through the Qt Image Formats plugin, so the plugin must be available at runtime on every target.

### Linux

On Ubuntu/Debian systems using the distribution Qt 6 packages, install the Qt 6 image-format plugins:

```sh
sudo apt install qt6-image-formats-plugins
```

This package supplies the WebP image plugin used by `QImage`/`QImageReader`.

### Windows

When packaging the cross-built Windows client, include the WebP image plugin from the Windows Qt installation alongside the other Qt plugins.

## Transport security

The Qt client uses Qt Network for all HTTP API requests. When the server is configured with TLS and the client uses an `https://` server URL, the login, authentication token, Home content, chat, buddy/presence, messages, events, and other HTTP API traffic are encrypted by TLS in transit.

The client does **not** disable certificate validation. The server certificate must be trusted by the operating system and must match the hostname used in the server URL. For local development without a trusted certificate, the existing `http://127.0.0.1:9090` mode remains available.

The separate native JSON-lines listener (`:8023`) and OSCAR compatibility listener (`:5190`) are not changed by this HTTP TLS slice. Their transport security/compatibility requirements are a separate concern.
