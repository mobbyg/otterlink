# OtterLink — Future Games & Interactive Content Roadmap

> **Status: Future planning / ideas only**
>
> This document describes possible future directions for OtterLink's games, interactive fiction, and community-created content. It is intentionally a roadmap rather than an implementation plan for the current development cycle.

## Overview

One possible future direction for OtterLink is to add an interactive games and entertainment layer that combines modern web-based games with classic BBS door-game and text-adventure concepts.

The goal is to use modern, open technologies to recreate the feeling of classic BBS games, doors, and text adventures while allowing OtterLink to interact with them through its Qt interface.

An important long-term goal is to make OtterLink a **platform for games created by the community**, rather than requiring the OtterLink developers to create every game themselves.

---

## 1. The OtterLink Game API

The central idea for the future game system should be a common **OtterLink Game API**.

Rather than building a special integration for every game, OtterLink would provide a documented interface that games can use to communicate with the OtterLink server and client.

The game should not need to understand the internal workings of OtterLink. Instead:

```text
Game
  |
  | OtterLink Game API
  v
OtterLink Server
  |
  v
OtterLink Client
```

The API could eventually provide:

- Player identification
- Authentication
- Game sessions
- Player state
- Persistent game data
- Multiplayer communication
- Messaging
- Chat
- Scores and high-score tables
- Game saves
- Game lifecycle events
- Game configuration
- Permissions
- Version information

The API should be simple enough that an independent developer can create an OtterLink game without needing to understand the entire OtterLink codebase.

---

## 2. Multiple Game Platforms

The Game API should ideally support several different types of games.

```text
                         OtterLink Game API
                                |
          +---------------------+---------------------+
          |                     |                     |
     Web/JavaScript          Pygame             Text/IF
          |                     |                     |
      HTML5/Canvas           Python          Z-Machine /
                                             Inform 7
          |                     |                     |
          +---------------------+---------------------+
                                |
                         OtterLink Client
                              Qt/Web
```

This gives developers choices rather than forcing everything into one technology.

Potential supported approaches include:

- HTML5 / JavaScript games
- Python / Pygame games
- Text-based games
- Interactive fiction
- Z-Machine / Z-code games
- Inform 7 adventures
- MUD-style applications
- Other open-source game engines where practical

---

## 3. Web-Based Board Games

A natural starting point would be simple multiplayer games implemented with HTML5 and JavaScript.

Possible initial games:

- Chess
- Checkers
- Backgammon
- Tic-Tac-Toe
- Connect Four
- Reversi/Othello
- Battleship
- Other simple turn-based games

The games could remain relatively independent from OtterLink while OtterLink provides:

- User authentication
- Player identity
- Game launching
- Multiplayer communication
- Game/session management
- Chat
- Scores and statistics
- Game history

Where practical, existing open-source JavaScript games could be integrated rather than developing every game from scratch.

---

## 4. Pygame Games

**Pygame should be a first-class development option for OtterLink games.**

Pygame is attractive because it provides an accessible way for developers familiar with Python to create games without having to learn the entire OtterLink or Qt architecture.

Possible Pygame games could include:

- Arcade games
- Puzzle games
- Card games
- Strategy games
- RPGs
- Retro games
- Simple graphical adventures
- Turn-based games
- Multiplayer games

A future **OtterLink Python SDK** could provide a wrapper around the Game API.

Conceptually:

```text
Python/Pygame Game
        |
        v
OtterLink Python SDK
        |
        v
OtterLink Game API
        |
        v
OtterLink Server
```

The exact API would be designed later, but could eventually provide simple operations for things such as:

- Getting the current player
- Starting and ending games
- Saving and loading game state
- Sending messages
- Getting players in a session
- Updating scores
- Storing player data

### Community Development

This could eventually allow an OtterLink user to download the SDK, learn the API, create a game in Pygame, and submit it for inclusion in an OtterLink installation.

This changes the role of OtterLink from simply **running games** to providing a platform where the community can **create games for OtterLink**.

---

## 5. OtterLink SDK

As the Game API matures, an official SDK could be provided for supported development environments.

Potential SDKs:

```text
OtterLink SDK
├── Python
│   └── Pygame support
├── JavaScript
│   └── HTML5/Web games
└── Documentation
    ├── Game API
    ├── Examples
    ├── Packaging
    └── Publishing
```

Example projects could include:

- Checkers
- Acrophobia
- A small Pygame arcade game
- A text adventure
- A multiplayer demonstration
- A simple RPG

The examples would serve both as tests of the API and as starting points for people wanting to create their own games.

---

## 6. Classic Text Adventures

Another major possibility is support for classic-style interactive fiction.

The obvious inspiration is the Infocom era, including games such as Zork and Planetfall.

Rather than attempting to recreate those games directly, OtterLink could provide an environment capable of launching compatible interactive-fiction engines.

### Inform 7

Inform 7 is particularly interesting because it provides a relatively accessible way to create completely new interactive fiction.

A possible future workflow:

```text
Author creates adventure
        |
        v
Inform 7
        |
        v
Compile
        |
        v
Interactive-fiction format
        |
        v
OtterLink
        |
        v
Player
```

This could allow OtterLink users to create **new adventures specifically for their own BBS/community**.

---

## 7. Z-Machine / ZIL / Classic Adventure Support

There is also an opportunity to support older Infocom technology more directly.

Potential formats and technologies to investigate:

- Z-Machine
- Z-code
- ZIL
- Existing open-source interpreters
- Modern interactive-fiction interpreters

A future OtterLink "door" could essentially be a wrapper around an interpreter.

This would allow classic text adventures and newly created interactive fiction to fit into the same general game ecosystem.

---

## 8. Classic BBS Games

OtterLink could eventually support games inspired by traditional BBS doors.

Examples include:

- Trivia
- Word games
- Strategy games
- Card games
- RPGs
- Trading/economic games
- Turn-based multiplayer games
- ASCII/ANSI games
- Simple graphical games

The goal would not necessarily be to emulate every historical door protocol. Instead, OtterLink could provide a **modern door abstraction** that gives older-style programs a common way to interact with the system.

---

## 9. Acrophobia-Style Games

A particularly fun possibility is bringing back games in the style of **Acrophobia — The Fear of Acronyms**.

The basic concept works extremely well for a modern networked application:

1. System presents an acronym.
2. Players invent meanings for it.
3. Players submit their answers.
4. Answers are displayed anonymously.
5. Players vote.
6. Scores are calculated.
7. The next round begins.

This could work through:

- Qt-native UI
- Embedded web UI
- A hybrid approach

It could also provide an excellent early example of how a classic BBS game concept can be modernized while retaining its original character.

---

## 10. Simple Graphics

Not everything needs to be purely text-based.

A future OtterLink game environment could support simple graphics while retaining the aesthetic of classic BBS software.

Possible styles include:

- ANSI art
- ASCII art
- Pixel graphics
- Simple 2D HTML5 graphics
- Canvas-based games
- Retro UI
- Low-resolution graphics
- Modern graphics with retro presentation

This could become a distinctive OtterLink aesthetic:

**Classic BBS personality + modern technology.**

---

## 11. MUD / Multiplayer Worlds

A larger long-term possibility would be support for a MUD or MUD-like environment.

Potential features could include:

- Persistent characters
- Rooms and areas
- NPCs
- Combat
- Inventory
- Quests
- Economy
- Chat
- Player interaction
- Persistent world state

A MUD could initially be completely text-based. Later, the Qt interface could add graphical elements around the text.

---

## 12. Community Game Library

Eventually, OtterLink could have its own **Game Library**.

A sysop could install games into an OtterLink instance and make them available to users.

For example:

```text
OTTERLINK GAME LIBRARY

  1. Chess
  2. Checkers
  3. Backgammon
  4. Acrophobia
  5. Space Trader
  6. Dungeon Adventure
  7. [User Submitted] Galactic Wars
  8. [User Submitted] OtterQuest

  A. Add Game
  R. Remove Game
  U. Update Game
```

Games could carry metadata such as:

```text
Name:          OtterQuest
Author:        Joe Smith
Version:       1.2
Engine:        Pygame
Players:       1-4
Category:      RPG
License:       GPL-3.0
OtterLink API: 1.x
```

This could eventually become a small ecosystem of OtterLink-compatible games.

---

## 13. Game Packaging and Installation

A future game package could contain everything necessary to install a game into an OtterLink instance.

Conceptually:

```text
otterquest.otgame
|
├── manifest
├── game/
├── assets/
├── documentation/
└── license
```

The manifest could tell OtterLink:

- Game name
- Author
- Version
- Required API version
- Engine/runtime
- Number of players
- Permissions
- Required resources
- License

The exact package format should be determined later.

---

## 14. Open-Source First

A strong guiding principle should be:

> **Don't reinvent an existing open-source game engine if a good one already exists.**

Where practical, OtterLink should integrate existing open-source projects and engines.

That could include:

- JavaScript games
- HTML5 games
- Pygame projects
- Interactive-fiction interpreters
- Z-Machine interpreters
- MUD engines
- Existing BBS/door software
- Other open game engines

OtterLink's job would primarily be to provide the integration layer and user experience.

This could dramatically increase the amount of content available without making OtterLink responsible for maintaining every game itself.

---

## 15. Suggested Development Path

When this eventually becomes an active project, a sensible progression would be:

### Phase 1 — Define the Game API

Before building a large collection of games, establish the basic contract between a game and OtterLink.

Start with:

- Player identity
- Game lifecycle
- Sessions
- Messaging
- Basic persistent data
- Scores

### Phase 2 — Simple Web Game

Start with something extremely simple such as **Checkers**.

Use it to prove that OtterLink can:

- Launch a web-based game
- Display it through Qt
- Identify users
- Handle game sessions
- Communicate with the server

### Phase 3 — Pygame Prototype

Create a small Pygame game using the OtterLink Python SDK.

This proves that the API is not tied to the browser and establishes the beginning of a developer SDK.

### Phase 4 — More Games

Add:

- Chess
- Backgammon
- Other simple turn-based games
- Additional Pygame examples

At this point, refine the reusable game integration framework.

### Phase 5 — Interactive Fiction

Add support for a text-adventure interpreter.

Investigate:

- Inform 7
- Z-Machine
- Z-code
- Other open interactive-fiction engines

### Phase 6 — BBS Door Compatibility

Develop a generalized door interface and begin integrating classic-style games.

### Phase 7 — Community-Created Content

Allow users to install/add their own:

- Games
- Adventures
- Interactive fiction
- Doors

Provide documentation and example projects so developers can build their own content.

### Phase 8 — Community Game Library

Develop a standardized package format and local game library/catalog.

### Phase 9 — Multiplayer / MUD

Investigate integrating or developing a MUD-style persistent multiplayer environment.

---

## Guiding Philosophy

The overall objective is **not simply "add games to OtterLink."**

The bigger idea is to create a modern platform where:

**BBS → Door Games → Interactive Fiction → Web Games → Pygame → Multiplayer Worlds**

can all coexist.

OtterLink could effectively become a modern interpretation of the old BBS experience:

> **A communications platform where the games are part of the community rather than merely applications launched from a menu.**

And, perhaps most importantly:

> **OtterLink should eventually make it possible for its users to become game developers and content creators, not just players.**
