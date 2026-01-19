# Setback (Pitch)

A 4-player, 2-team online card game built with Go and vanilla JavaScript.

## Quick Start

```bash
# Run the server
go run ./cmd/server

# Open browser to http://localhost:8080
```

## Command Line Options

```bash
go run ./cmd/server -port 8080 -target 52
```

- `-port`: Server port (default: 8080)
- `-target`: Target score to win (default: 52)

## How to Play

1. Open 4 browser tabs to http://localhost:8080
2. Each player enters their name and clicks a seat button
3. Once all 4 seats are filled, click "Start Game"
4. **Bidding**: Players bid 2-4 or pass. High bidder names trump.
5. **Playing**: Play 6 tricks. Follow suit if able, or play trump.
6. **Scoring**: Points for High, Low, Jack, and Game.

## Game Rules

### Teams
- Seats 1 & 3 vs Seats 2 & 4

### Bidding
- Bid 2, 3, or 4, or pass
- Must bid higher than previous bid
- If all pass, dealer takes minimum bid (2)
- High bidder leads first trick; their first card sets trump

### Playing
- Must follow lead suit if able
- Can play trump at any time
- Highest trump wins, else highest of lead suit

### Scoring (per hand)
- **High**: 1 point to team with highest trump played
- **Low**: 1 point to team with lowest trump played
- **Jack**: 1 point to team capturing Jack of trump
- **Game**: 1 point to team with most game points (A=4, K=3, Q=2, J=1, 10=10)

### Setback
If the bidding team doesn't make their bid, they lose bid points instead of gaining.

## Project Structure

```
setback/
├── cmd/server/main.go   # Entry point
├── game/
│   ├── card.go          # Card, Deck types
│   ├── state.go         # GameState, Phase, Player
│   ├── engine.go        # State machine logic
│   └── scoring.go       # Score calculation
├── server/
│   ├── hub.go           # WebSocket hub
│   ├── protocol.go      # Message types
│   └── handlers.go      # Action handlers
├── static/
│   ├── index.html       # UI
│   ├── app.js           # Frontend logic
│   ├── style.css        # Styling
│   └── cards/           # Card SVGs
└── go.mod
```

## Development

```bash
# Run tests
go test ./...

# Build binary
go build -o setback ./cmd/server
```
