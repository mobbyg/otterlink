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

## Direction

This is a native client foundation, not a finished UI. Service operations should remain behind the client/service boundary so the same Otter Link account and community can later be presented by native Amiga/AROS, C64/C128, and Commander X16 clients.
