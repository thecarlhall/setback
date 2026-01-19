# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Setback (Pitch) is a 4-player, 2-team online card game. Teams are fixed: seats 0+2 vs seats 1+3. The goal is a clean, maintainable codebase with real-time browser gameplay via WebSockets.

## Build & Run Commands

Once implemented, the project should support:

```bash
# Backend (Go)
go run ./cmd/server

# Run tests
go test ./...
```

## Architecture

**Backend** (use Go):
- Pure state machine design: state in, action in, new state out
- WebSocket endpoint at `/ws` for real-time communication
- Server is authoritative - validates all moves

**Frontend** (HTML/JS with React, Vue, Svelte, or vanilla):
- Single-page app, no page refresh during play
- Updates only via WebSocket messages

**Package structure**:
- `game/` - Core game logic (bidding, trick play, scoring)
- `server/` - WebSocket handling, session management
- `model/` - Domain entities (Card, Deck, Player, Team, Trick, Bid, GameState, Phase)

## Game Rules Reference

Source of truth for rules:
- Main rules: https://www.singaporemahjong.com/pitch/rules/
- Background: https://en.wikipedia.org/wiki/Pitch_(card_game)

Key rules:
- Standard 52-card deck, 6 cards dealt per player
- Bidding: Pass or bid 2-4; dealer forced to take minimum if all pass
- Scoring points: High, Low, Jack, Off-Jack (other suit of same color) Game (A=4, K=3, Q=2, J=1, 10=10)
- Setback penalty: bidding team loses bid amount if they fail to make it
- Game ends at configurable target score (default 52)

## WebSocket Protocol

Client → Server:
- `joinTable { seatIndex, playerName }`
- `placeBid { amount | "pass" }`
- `playCard { cardId }`
- `rejoin { sessionToken }`

Server → Client:
- `stateUpdate { publicState, yourPrivateState }`
- `error { code, message }`

## Configuration

Use environment variables or config file for:
- Port numbers
- Target score (default 52)
- Allowed bid range

## Testing Focus

Tests should cover:
- Bidding logic and validation
- Trick resolution (follow-suit rules, trump behavior)
- Scoring calculation (High, Low, Jack, Game, setback penalties)

## Style Guidelines

- Favor explicit, straightforward code over clever abstractions
- Use explicit types and enums, avoid magic strings
- Keep rule implementations in well-named functions with comments linking to rules pages
- Design for single table but structure code to be extendable
