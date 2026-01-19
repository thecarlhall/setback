package server

import "setback/game"

// MessageType identifies the type of WebSocket message
type MessageType string

const (
	// Client -> Server messages
	MsgJoinTable    MessageType = "joinTable"
	MsgLeaveSeat    MessageType = "leaveSeat"
	MsgChangeName   MessageType = "changeName"   // Change player name
	MsgKickPlayer     MessageType = "kickPlayer"     // House only: kick a player
	MsgTransferHouse  MessageType = "transferHouse"  // House only: transfer house to another player
	MsgStartGame      MessageType = "startGame"
	MsgPlaceBid     MessageType = "placeBid"
	MsgSelectTrump  MessageType = "selectTrump"  // Kitty phase: select trump suit
	MsgTakeKitty    MessageType = "takeKitty"    // Kitty phase: take cards from kitty
	MsgDiscard      MessageType = "discard"      // Kitty phase: bid winner discards to 6
	MsgDiscardDraw  MessageType = "discardDraw"  // Discard phase: discard and draw replacements
	MsgPlayCard     MessageType = "playCard"
	MsgRejoin       MessageType = "rejoin"
	MsgNewHand      MessageType = "newHand"
	MsgResetGame    MessageType = "resetGame"    // Admin only: reset game to lobby

	// Server -> Client messages
	MsgStateUpdate  MessageType = "stateUpdate"
	MsgError        MessageType = "error"
	MsgScoreUpdate  MessageType = "scoreUpdate"
	MsgGameOver     MessageType = "gameOver"
)

// ClientMessage represents a message from client to server
type ClientMessage struct {
	Type       MessageType `json:"type"`
	SeatIndex  *int        `json:"seatIndex,omitempty"`
	PlayerName string      `json:"playerName,omitempty"`
	Amount     *int        `json:"amount,omitempty"` // Bid amount (0 = pass)
	CardID     string      `json:"cardId,omitempty"`
	CardIDs    []string    `json:"cardIds,omitempty"`  // For taking/discarding multiple cards
	TrumpSuit  string      `json:"trumpSuit,omitempty"` // For selecting trump
	Token      string      `json:"token,omitempty"`     // Session token for rejoin
}

// ServerMessage represents a message from server to client
type ServerMessage struct {
	Type         MessageType       `json:"type"`
	State        *PublicState      `json:"state,omitempty"`
	YourHand     []game.Card       `json:"yourHand,omitempty"`
	Kitty        []game.Card       `json:"kitty,omitempty"` // Shown to bid winner during kitty phase
	YourSeat     *int              `json:"yourSeat,omitempty"`
	YourToken    string            `json:"yourToken,omitempty"`
	Error        *ErrorPayload     `json:"error,omitempty"`
	ScoreResult  *game.ScoreResult `json:"scoreResult,omitempty"`
	WinningTeam  *int              `json:"winningTeam,omitempty"`
}

// ErrorPayload contains error information
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PublicState is the game state visible to all players
type PublicState struct {
	Phase         game.Phase     `json:"phase"`
	Players       []PublicPlayer `json:"players"`
	Teams         [2]TeamState   `json:"teams"`
	CurrentTrick  *TrickState    `json:"currentTrick"`
	LastTrick     *TrickState    `json:"lastTrick"` // Previous trick for display
	Trump         *string        `json:"trump"`
	Bids          []game.Bid     `json:"bids"`
	Dealer        int            `json:"dealer"`
	CurrentPlayer int            `json:"currentPlayer"`
	TricksPlayed  int            `json:"tricksPlayed"`
	BidWinner     int            `json:"bidWinner"`
	WinningBid    int            `json:"winningBid"`
	TargetScore   int            `json:"targetScore"`
	KittyCount    int            `json:"kittyCount"` // Number of cards in kitty
	House         int            `json:"house"`      // Seat index of the house (game owner)
	TrumpBroken   bool           `json:"trumpBroken"` // Whether trump has been played this hand
}

// PublicPlayer is player info visible to all
type PublicPlayer struct {
	Name            string `json:"name"`
	SeatIndex       int    `json:"seatIndex"`
	Connected       bool   `json:"connected"`
	CardCount       int    `json:"cardCount"` // Number of cards in hand
	HasBid          bool   `json:"hasBid"`
	DiscardReady    bool   `json:"discardReady"`    // Has submitted discard selection (waiting for turn)
	DiscardComplete bool   `json:"discardComplete"` // Has completed discard and draw
}

// TeamState is team info visible to all
type TeamState struct {
	PlayerIndices []int `json:"playerIndices"`
	Score         int   `json:"score"`
	GamesWon      int   `json:"gamesWon"`
}

// TrickState is the current trick visible to all
type TrickState struct {
	Cards    []TrickCardState `json:"cards"`
	Leader   int              `json:"leader"`
	LeadSuit string           `json:"leadSuit"`
	Winner   int              `json:"winner"` // Set when trick is complete
}

// TrickCardState is a played card visible to all
type TrickCardState struct {
	Card        game.Card `json:"card"`
	PlayerIndex int       `json:"playerIndex"`
}

// BuildPublicState creates the public state from game state
func BuildPublicState(gs *game.GameState) *PublicState {
	ps := &PublicState{
		Phase:         gs.Phase,
		Players:       make([]PublicPlayer, 0, 4),
		Dealer:        gs.Dealer,
		CurrentPlayer: gs.CurrentPlayer,
		TricksPlayed:  gs.TricksPlayed,
		Bids:          gs.Bids,
		BidWinner:     gs.BidWinner,
		WinningBid:    gs.WinningBid,
		TargetScore:   gs.TargetScore,
		KittyCount:    len(gs.Kitty),
		House:         gs.House,
		TrumpBroken:   gs.TrumpBroken,
	}

	// Players
	for i, p := range gs.Players {
		if p == nil {
			ps.Players = append(ps.Players, PublicPlayer{
				SeatIndex: i,
				Connected: false,
			})
		} else {
			hasBid := false
			for _, b := range gs.Bids {
				if b.PlayerIndex == i {
					hasBid = true
					break
				}
			}
			ps.Players = append(ps.Players, PublicPlayer{
				Name:            p.Name,
				SeatIndex:       p.SeatIndex,
				Connected:       p.Connected,
				CardCount:       len(p.Hand),
				HasBid:          hasBid,
				DiscardReady:    gs.PendingDiscards[i] != nil,
				DiscardComplete: gs.DiscardComplete[i],
			})
		}
	}

	// Teams
	for i, t := range gs.Teams {
		ps.Teams[i] = TeamState{
			PlayerIndices: t.PlayerIndices,
			Score:         t.Score,
			GamesWon:      t.GamesWon,
		}
	}

	// Trump
	if gs.Trump != nil {
		trump := gs.Trump.String()
		ps.Trump = &trump
	}

	// Current trick
	if gs.CurrentTrick != nil {
		ps.CurrentTrick = &TrickState{
			Cards:    make([]TrickCardState, 0, len(gs.CurrentTrick.Cards)),
			Leader:   gs.CurrentTrick.Leader,
			LeadSuit: gs.CurrentTrick.LeadSuit.String(),
			Winner:   gs.CurrentTrick.Winner,
		}
		for _, tc := range gs.CurrentTrick.Cards {
			ps.CurrentTrick.Cards = append(ps.CurrentTrick.Cards, TrickCardState{
				Card:        tc.Card,
				PlayerIndex: tc.PlayerIndex,
			})
		}
	}

	// Last trick (for showing who won)
	if gs.LastTrick != nil {
		ps.LastTrick = &TrickState{
			Cards:    make([]TrickCardState, 0, len(gs.LastTrick.Cards)),
			Leader:   gs.LastTrick.Leader,
			LeadSuit: gs.LastTrick.LeadSuit.String(),
			Winner:   gs.LastTrick.Winner,
		}
		for _, tc := range gs.LastTrick.Cards {
			ps.LastTrick.Cards = append(ps.LastTrick.Cards, TrickCardState{
				Card:        tc.Card,
				PlayerIndex: tc.PlayerIndex,
			})
		}
	}

	return ps
}

// NewErrorMessage creates an error message
func NewErrorMessage(code, message string) ServerMessage {
	return ServerMessage{
		Type: MsgError,
		Error: &ErrorPayload{
			Code:    code,
			Message: message,
		},
	}
}

// NewStateUpdateMessage creates a state update message for a specific player
func NewStateUpdateMessage(gs *game.GameState, seatIndex int) ServerMessage {
	msg := ServerMessage{
		Type:  MsgStateUpdate,
		State: BuildPublicState(gs),
	}

	if seatIndex >= 0 && seatIndex < 4 && gs.Players[seatIndex] != nil {
		msg.YourHand = gs.Players[seatIndex].Hand
		msg.YourSeat = &seatIndex
		msg.YourToken = gs.Players[seatIndex].SessionToken

		// During kitty phase, show kitty to bid winner
		if gs.Phase == game.PhaseKitty && seatIndex == gs.BidWinner {
			msg.Kitty = gs.Kitty
		}
	}

	return msg
}
