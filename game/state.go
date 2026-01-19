package game

import (
	"crypto/rand"
	"encoding/hex"
)

// Phase represents the current game phase
type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhaseBidding  Phase = "bidding"
	PhaseKitty    Phase = "kitty"    // Bid winner selects trump and picks from kitty
	PhaseDiscard  Phase = "discard"  // Each player discards and draws replacements
	PhasePlaying  Phase = "playing"
	PhaseScoring  Phase = "scoring"
	PhaseFinished Phase = "finished"
)

// Player represents a player in the game
type Player struct {
	Name         string `json:"name"`
	SeatIndex    int    `json:"seatIndex"`
	Hand         []Card `json:"hand,omitempty"`
	SessionToken string `json:"-"`
	Connected    bool   `json:"connected"`
}

// GenerateSessionToken creates a random session token
func GenerateSessionToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Team represents a team of two players
type Team struct {
	PlayerIndices []int `json:"playerIndices"`
	Score         int   `json:"score"`
	GamesWon      int   `json:"gamesWon"` // Number of games (to target score) won
}

// Bid represents a bid made by a player
type Bid struct {
	PlayerIndex int `json:"playerIndex"`
	Amount      int `json:"amount"` // 0 = pass, 2-4 = bid
}

// TrickCard represents a card played in a trick with its player
type TrickCard struct {
	Card        Card `json:"card"`
	PlayerIndex int  `json:"playerIndex"`
}

// Trick represents the current trick being played
type Trick struct {
	Cards    []TrickCard `json:"cards"`
	Leader   int         `json:"leader"`
	LeadSuit Suit        `json:"leadSuit"`
	Winner   int         `json:"winner"` // Set after trick completes
}

// CompletedTrick stores a finished trick with its winner
type CompletedTrick struct {
	Cards  []TrickCard `json:"cards"`
	Winner int         `json:"winner"`
}

// GameState represents the complete state of a game
type GameState struct {
	Phase         Phase      `json:"phase"`
	Players       [4]*Player `json:"players"`
	Teams         [2]*Team   `json:"teams"`
	Deck          *Deck      `json:"-"`
	CurrentTrick  *Trick     `json:"currentTrick"`
	LastTrick     *Trick     `json:"lastTrick"` // Previous trick for display
	Trump         *Suit      `json:"trump"`
	Bids          []Bid      `json:"bids"`
	Dealer        int        `json:"dealer"`
	CurrentPlayer int        `json:"currentPlayer"`
	TricksPlayed  int        `json:"tricksPlayed"`
	BidWinner     int        `json:"bidWinner"`
	WinningBid    int        `json:"winningBid"`
	TargetScore   int        `json:"targetScore"`
	House         int        `json:"house"` // Seat index of the house (game owner), -1 if none

	// Kitty - dealt to center, bid winner picks from it
	Kitty []Card `json:"kitty"`

	// Track who has completed discard phase
	DiscardComplete [4]bool `json:"-"`

	// Track pending discards - players can pre-select cards while waiting
	PendingDiscards [4][]string `json:"-"`

	// Track completed tricks for scoring
	CompletedTricks []CompletedTrick `json:"-"`

	// Cards won by each team (for Game point calculation)
	CardsWon [2][]Card `json:"-"`

	// Track if trump has been played this hand (broken)
	TrumpBroken bool `json:"trumpBroken"`
}

// NewGameState creates a new game in lobby phase
func NewGameState(targetScore int) *GameState {
	return &GameState{
		Phase:       PhaseLobby,
		Players:     [4]*Player{},
		Teams: [2]*Team{
			{PlayerIndices: []int{0, 2}, Score: 0},
			{PlayerIndices: []int{1, 3}, Score: 0},
		},
		TargetScore: targetScore,
		House:       -1, // No house until first player joins
	}
}

// GetTeamForPlayer returns the team index (0 or 1) for a player
func (g *GameState) GetTeamForPlayer(playerIndex int) int {
	if playerIndex == 0 || playerIndex == 2 {
		return 0
	}
	return 1
}

// AllPlayersSeated returns true if all 4 seats are filled
func (g *GameState) AllPlayersSeated() bool {
	for _, p := range g.Players {
		if p == nil {
			return false
		}
	}
	return true
}

// ConnectedPlayerCount returns the number of connected players
func (g *GameState) ConnectedPlayerCount() int {
	count := 0
	for _, p := range g.Players {
		if p != nil && p.Connected {
			count++
		}
	}
	return count
}

// NextPlayer returns the next player index (wrapping around)
func NextPlayer(current int) int {
	return (current + 1) % 4
}

// PlayerAfterDealer returns the player to the left of dealer (first to bid/play)
func (g *GameState) PlayerAfterDealer() int {
	return NextPlayer(g.Dealer)
}
