# Copilot Instructions for Setback (Pitch)

This document guides AI coding agents working in this repository. It summarizes architecture, workflows, and project-specific conventions for maximum productivity.

## Architecture Overview
- **Backend (Go):** Implements a pure state machine for the Setback card game. All game logic is in `game/` (see `engine.go`, `state.go`, `scoring.go`).
- **Server:** WebSocket-based, entry at `cmd/server/main.go`. Handles real-time communication and session management (`server/`).
- **Frontend:** Static files in `static/` (vanilla JS, HTML, CSS). UI updates only via WebSocket messages.
- **Data Flow:** Clients send actions (join, bid, play) via WebSocket; server validates and updates state, then broadcasts new state.

## Key Files & Directories
- `cmd/server/main.go` — Server entry point
- `game/` — Core game logic (card, state, engine, scoring)
- `server/` — WebSocket hub, protocol, handlers
- `static/` — Frontend assets (app.js, index.html, style.css, cards/)

## Developer Workflows
- **Run server:** `go run ./cmd/server`
- **Run tests:** `go test ./...`
- **Build binary:** `go build -o setback ./cmd/server`
- **Config:** Use command-line flags (`-port`, `-target`) or environment variables for server settings

## Project Conventions
- **State machine:** All game state transitions are pure functions (see `engine.go`)
- **Explicit code:** Prefer clear, direct logic over abstraction; use explicit types/enums
- **Rule links:** Comment rule logic with links to [Pitch rules](https://www.singaporemahjong.com/pitch/rules/)
- **WebSocket protocol:**
  - Client→Server: `joinTable`, `placeBid`, `playCard`, `rejoin`
  - Server→Client: `stateUpdate`, `error`
- **Testing:** Focus on bidding, trick resolution, scoring, and setback penalties

## Integration & Extensibility
- Server is authoritative; all moves validated server-side
- Designed for single-table play, but code should be modular for future multi-table support

## References
- [README.md](../README.md) — Quick start, rules, structure
- [CLAUDE.md](../CLAUDE.md) — Additional agent guidance

---
For any unclear conventions or missing patterns, consult `README.md`, `CLAUDE.md`, or ask for clarification.
