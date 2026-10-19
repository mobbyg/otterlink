# Otter Link for AROS (64-bit)

This directory is the home of the **native AROS client**. It is a client only: the existing Go server remains authoritative for accounts, sessions, buddies, presence, messages, and services. There is no AROS server, proxy, or duplicate database.

## Status

The first commit establishes a small native Zune/Intuition application shell and a repeatable build entry point. It is **not yet a connected client**: login, HTTP/JSON handling, buddy retrieval, and TLS are not implemented in this scaffold.

The intended first milestone is to validate the target SDK and build a direct, encrypted connection to the existing Otter Link API. See [issue #54](https://github.com/mobbyg/otterlink/issues/54).

## Target environment

Target a 64-bit x86 AROS distribution and use a toolchain/SDK matching that distribution's ABI. ABIv11 is a candidate, not an assumption: verify the exact distribution and SDK before treating any produced binary as compatible.

AROS developer documentation:
- [AROS Developer Portal](https://developers.aros.org/)
- [Zune application development](https://developers.aros.org/documentation/zune-dev/zune-application-development.html)

The Zune GUI shell uses native AROS/MUI-compatible APIs. It does not depend on Qt.

## Build

From this directory, with the matching AROS cross-compiler and SDK available in `PATH`:

```sh
make
```

By default, the Makefile uses `x86_64-aros-gcc`. Override it if your SDK uses a different compiler executable:

```sh
make AROS_CC=/path/to/your/aros-gcc
```

The output is `otterlink-aros`. Copy it to the target AROS system and run it there. The build has not yet been verified against a specific AROS distribution; report compiler or SDK differences in issue #54 so the documented toolchain can be pinned.

Clean generated output with:

```sh
make clean
```

## Planned implementation sequence

1. Confirm the exact AROS distribution, ABI, and SDK.
2. Verify HTTP, JSON, and TLS options available in that environment.
3. Add a small connection test against the existing API, with encrypted transport required before production credentials are sent.
4. Implement login and current-user/buddy retrieval.
5. Grow the Zune interface into the buddy list, presence, private messages, and Community Chat.

The API contract is documented in [`docs/api.md`](../../docs/api.md). The separate native TCP/JSON protocol is still a draft, so the first proof of concept should prefer the HTTP API unless investigation supports another choice.
