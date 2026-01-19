package game

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
)

// Suit represents a card suit
type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

func (s Suit) String() string {
	return [...]string{"spades", "hearts", "diamonds", "clubs"}[s]
}

// AllSuits returns all suits in order
func AllSuits() []Suit {
	return []Suit{Spades, Hearts, Diamonds, Clubs}
}

// OffSuit returns the other suit of the same color
// Spades <-> Clubs (black), Hearts <-> Diamonds (red)
func (s Suit) OffSuit() Suit {
	switch s {
	case Spades:
		return Clubs
	case Clubs:
		return Spades
	case Hearts:
		return Diamonds
	case Diamonds:
		return Hearts
	default:
		return s
	}
}

// Rank represents a card rank (2-14, where 11=J, 12=Q, 13=K, 14=A)
type Rank int

const (
	Two   Rank = 2
	Three Rank = 3
	Four  Rank = 4
	Five  Rank = 5
	Six   Rank = 6
	Seven Rank = 7
	Eight Rank = 8
	Nine  Rank = 9
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
	Ace   Rank = 14
)

func (r Rank) String() string {
	switch r {
	case Jack:
		return "jack"
	case Queen:
		return "queen"
	case King:
		return "king"
	case Ace:
		return "ace"
	default:
		return fmt.Sprintf("%d", r)
	}
}

// AllRanks returns all ranks in order (2-A)
func AllRanks() []Rank {
	return []Rank{Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}
}

// GamePoints returns the game point value for scoring
// A=4, K=3, Q=2, J=1, 10=10, all others=0
func (r Rank) GamePoints() int {
	switch r {
	case Ace:
		return 4
	case King:
		return 3
	case Queen:
		return 2
	case Jack:
		return 1
	case Ten:
		return 10
	default:
		return 0
	}
}

// Card represents a playing card
type Card struct {
	ID   string `json:"id"`
	Suit Suit   `json:"suit"`
	Rank Rank   `json:"rank"`
}

// NewCard creates a new card with auto-generated ID
func NewCard(suit Suit, rank Rank) Card {
	return Card{
		ID:   fmt.Sprintf("%s_%s", rank.String(), suit.String()),
		Suit: suit,
		Rank: rank,
	}
}

// IsTrump returns true if this card is trump (including the Off Jack)
func (c Card) IsTrump(trump Suit) bool {
	if c.Suit == trump {
		return true
	}
	// Off Jack (same color Jack) is also trump
	if c.Rank == Jack && c.Suit == trump.OffSuit() {
		return true
	}
	return false
}

// IsOffJack returns true if this card is the Off Jack for the given trump
func (c Card) IsOffJack(trump Suit) bool {
	return c.Rank == Jack && c.Suit == trump.OffSuit()
}

// TrumpRank returns the effective rank for trump comparison
// Off Jack ranks just below Jack of trump (between Jack and 10)
// Returns a float to allow Off Jack (10.5) to slot between Jack (11) and 10
func (c Card) TrumpRank(trump Suit) float64 {
	if c.IsOffJack(trump) {
		return 10.5 // Between Jack (11) and Ten (10)
	}
	return float64(c.Rank)
}

// Beats returns true if this card beats the other card given the trump suit and lead suit
func (c Card) Beats(other Card, trump Suit, leadSuit Suit) bool {
	cIsTrump := c.IsTrump(trump)
	otherIsTrump := other.IsTrump(trump)

	// Trump always beats non-trump
	if cIsTrump && !otherIsTrump {
		return true
	}
	if !cIsTrump && otherIsTrump {
		return false
	}

	// Both trump - compare using trump rank (handles Off Jack)
	if cIsTrump && otherIsTrump {
		return c.TrumpRank(trump) > other.TrumpRank(trump)
	}

	// Neither trump - same suit comparison
	if c.Suit == other.Suit {
		return c.Rank > other.Rank
	}

	// Different non-trump suits - lead suit wins
	if c.Suit == leadSuit {
		return true
	}
	return false
}

// Deck represents a deck of cards
type Deck struct {
	Cards []Card
}

// NewDeck creates a standard 52-card deck
func NewDeck() *Deck {
	d := &Deck{Cards: make([]Card, 0, 52)}
	for _, suit := range AllSuits() {
		for _, rank := range AllRanks() {
			d.Cards = append(d.Cards, NewCard(suit, rank))
		}
	}
	return d
}

// Shuffle randomizes the deck order using cryptographically secure randomness
func (d *Deck) Shuffle() {
	// Seed math/rand with cryptographically secure random bytes
	var seed int64
	binary.Read(rand.Reader, binary.LittleEndian, &seed)
	rng := mathrand.New(mathrand.NewSource(seed))

	rng.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Deal removes and returns n cards from the top of the deck
// Returns a copy of the cards to prevent slice aliasing issues
func (d *Deck) Deal(n int) []Card {
	if n > len(d.Cards) {
		n = len(d.Cards)
	}
	// Create a new slice with copies of the cards
	dealt := make([]Card, n)
	copy(dealt, d.Cards[:n])
	d.Cards = d.Cards[n:]
	return dealt
}

// Remaining returns how many cards are left
func (d *Deck) Remaining() int {
	return len(d.Cards)
}
