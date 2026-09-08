# Otter Link Presentation Direction

Otter Link is a modern online service with a deliberately retro presentation layer. The service engine, account system, networking, persistence, and security should remain modern; the client presentation can evoke the connected-computer experience of the 1980s and 1990s.

## Visual direction

The primary modern desktop client should take inspiration from the visual language of classic Quantum Link / Q-Link / AOL-era online services without becoming a pixel-for-pixel recreation of any one product.

The goal is **retro feel, modern engine**:

- period-inspired windows, panels, controls, icons, dividers, and artwork
- Otter Link branding rather than copied branding
- usable modern window behavior and accessibility
- presentation implemented in the client, not in the server or service model
- room for platform-specific presentation on Amiga/AROS, C64/C128, and Commander X16

## Connection experience

The desktop client should eventually include an optional dial-up-style connection sequence. It is a presentation of the connection process, not an artificial delay in the actual network connection.

Conceptually the sequence is:

1. **Calling** — the client begins connecting and presents Otter Link connection artwork.
2. **Connecting** — the client presents the modem/carrier phase and corresponding otter artwork.
3. **Connected** — the client transitions into the main service UI.

The underlying connection should proceed normally while the presentation layer follows the real connection state.

## Connection audio

The client should support optional connection and disconnect audio, including a period-inspired telephone/modem/carrier sequence. WAV and MP3 assets may be supported where appropriate for the platform.

Audio should be independently controllable from the visual connection experience. A user should be able to:

- enable/disable the connection animation
- enable/disable connection sounds
- enable/disable disconnect sounds
- control connection/notification volume

The default behavior can provide the nostalgic experience while allowing users who simply want to connect quickly to turn individual effects off.

## Artwork

Artwork should be original Otter Link material or appropriately licensed material. Historical software and screenshots may be used as design references, but the project should not depend on redistributing copyrighted legacy assets without permission.

The otter becomes the visual mascot for the connection experience and can later have contextual artwork for areas such as mail, buddies, messages, errors, and events.

Initial artwork concepts:

- calling otter
- connecting/carrier otter
- connected otter
- disconnected/offline otter
- mail/message otter
- buddy/presence otter
- error/problem otter

## Theme architecture

The retro presentation should be treated as a client-side theme/presentation layer rather than hard-coded into the service.

A future Qt client can support multiple presentations over the same service model, for example:

- **Otter Link Classic** — full 1990s online-service aesthetic
- **Otter Link Modern** — restrained contemporary presentation
- **Otter Link Dark** — modern dark presentation

This also keeps the architecture aligned with the project's larger goal: one service, many clients, with the machine and client determining how the service feels.

## Qt implementation direction

Qt Designer `.ui` files should remain the preferred way to define substantial desktop layouts. C++ should handle service behavior, state, and interaction rather than becoming a monolithic hand-built UI.

The connection experience should eventually be implemented as a reusable presentation component so it can be invoked during login without coupling the service client to specific artwork or sound assets.
