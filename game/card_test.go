package game

import (
	"testing"
)

func TestNewDeck(t *testing.T) {
	deck := NewDeck()

	// Should have exactly 52 cards
	if len(deck.Cards) != 52 {
		t.Errorf("Expected 52 cards, got %d", len(deck.Cards))
	}

	// All cards should be unique
	seen := make(map[string]bool)
	for _, card := range deck.Cards {
		if seen[card.ID] {
			t.Errorf("Duplicate card found: %s", card.ID)
		}
		seen[card.ID] = true
	}

	// Should have 13 cards per suit
	suitCounts := make(map[Suit]int)
	for _, card := range deck.Cards {
		suitCounts[card.Suit]++
	}
	for _, suit := range AllSuits() {
		if suitCounts[suit] != 13 {
			t.Errorf("Expected 13 cards for suit %s, got %d", suit, suitCounts[suit])
		}
	}

	// Should have 4 cards per rank
	rankCounts := make(map[Rank]int)
	for _, card := range deck.Cards {
		rankCounts[card.Rank]++
	}
	for _, rank := range AllRanks() {
		if rankCounts[rank] != 4 {
			t.Errorf("Expected 4 cards for rank %s, got %d", rank, rankCounts[rank])
		}
	}
}

func TestDeckShuffle(t *testing.T) {
	deck1 := NewDeck()
	deck2 := NewDeck()

	deck1.Shuffle()
	deck2.Shuffle()

	// After shuffling, decks should (almost certainly) be different
	sameOrder := true
	for i := range deck1.Cards {
		if deck1.Cards[i].ID != deck2.Cards[i].ID {
			sameOrder = false
			break
		}
	}

	if sameOrder {
		t.Error("Two shuffled decks have identical order - shuffle may not be working")
	}

	// Each shuffled deck should still have 52 unique cards
	for _, deck := range []*Deck{deck1, deck2} {
		seen := make(map[string]bool)
		for _, card := range deck.Cards {
			if seen[card.ID] {
				t.Errorf("Duplicate card found after shuffle: %s", card.ID)
			}
			seen[card.ID] = true
		}
	}
}

func TestDeckDeal(t *testing.T) {
	deck := NewDeck()
	deck.Shuffle()

	// Deal cards to 4 players and kitty (6 each = 30 cards)
	hands := make([][]Card, 4)
	for i := 0; i < 4; i++ {
		hands[i] = deck.Deal(6)
		if len(hands[i]) != 6 {
			t.Errorf("Expected 6 cards dealt, got %d", len(hands[i]))
		}
	}
	kitty := deck.Deal(6)

	// Should have 22 cards remaining
	if deck.Remaining() != 22 {
		t.Errorf("Expected 22 cards remaining, got %d", deck.Remaining())
	}

	// All dealt cards should be unique
	seen := make(map[string]bool)
	for i, hand := range hands {
		for _, card := range hand {
			if seen[card.ID] {
				t.Errorf("Duplicate card found in hand %d: %s", i, card.ID)
			}
			seen[card.ID] = true
		}
	}
	for _, card := range kitty {
		if seen[card.ID] {
			t.Errorf("Duplicate card found in kitty: %s", card.ID)
		}
		seen[card.ID] = true
	}

	// Add remaining deck cards
	for _, card := range deck.Cards {
		if seen[card.ID] {
			t.Errorf("Duplicate card found in remaining deck: %s", card.ID)
		}
		seen[card.ID] = true
	}

	// Total unique cards should be 52
	if len(seen) != 52 {
		t.Errorf("Expected 52 unique cards total, got %d", len(seen))
	}
}

func TestAllRanks(t *testing.T) {
	ranks := AllRanks()
	if len(ranks) != 13 {
		t.Errorf("Expected 13 ranks, got %d", len(ranks))
	}

	// Verify order: 2, 3, 4, 5, 6, 7, 8, 9, 10, J, Q, K, A
	expected := []Rank{Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}
	for i, rank := range ranks {
		if rank != expected[i] {
			t.Errorf("Rank %d: expected %v, got %v", i, expected[i], rank)
		}
	}
}

func TestAllSuits(t *testing.T) {
	suits := AllSuits()
	if len(suits) != 4 {
		t.Errorf("Expected 4 suits, got %d", len(suits))
	}

	// Verify all suits present
	expected := map[Suit]bool{Spades: true, Hearts: true, Diamonds: true, Clubs: true}
	for _, suit := range suits {
		if !expected[suit] {
			t.Errorf("Unexpected suit: %v", suit)
		}
		delete(expected, suit)
	}
	if len(expected) > 0 {
		t.Errorf("Missing suits: %v", expected)
	}
}
