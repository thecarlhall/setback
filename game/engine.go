package game

import (
	"errors"
	"fmt"
)

// Action types
type ActionType string

const (
	ActionJoinSeat    ActionType = "joinSeat"
	ActionLeaveSeat   ActionType = "leaveSeat"
	ActionChangeName  ActionType = "changeName"
	ActionKickPlayer    ActionType = "kickPlayer"    // House only: kick a player from their seat
	ActionTransferHouse ActionType = "transferHouse" // House only: transfer house to another player
	ActionStartGame   ActionType = "startGame"
	ActionPlaceBid    ActionType = "placeBid"
	ActionSelectTrump ActionType = "selectTrump"
	ActionTakeKitty   ActionType = "takeKitty"   // Take cards from kitty into hand
	ActionDiscard     ActionType = "discard"     // Bid winner discards to get to 6
	ActionDiscardDraw ActionType = "discardDraw" // Any player discards and draws replacements
	ActionPlayCard    ActionType = "playCard"
	ActionResetGame   ActionType = "resetGame"   // House only: reset game to lobby
)

// Action represents a game action
type Action struct {
	Type        ActionType
	PlayerIndex int
	PlayerName  string
	BidAmount   int
	CardID      string
	TrumpSuit   string   // For SelectTrump action
	CardIDs     []string // For TakeKitty/Discard actions (multiple cards)
	TargetSeat  int      // For KickPlayer action
}

// Common errors
var (
	ErrNotYourTurn      = errors.New("not your turn")
	ErrInvalidAction    = errors.New("invalid action for current phase")
	ErrSeatTaken        = errors.New("seat already taken")
	ErrSeatEmpty        = errors.New("seat is empty")
	ErrNotEnoughPlayers = errors.New("need 4 players to start")
	ErrInvalidBid       = errors.New("invalid bid amount")
	ErrCardNotInHand    = errors.New("card not in hand")
	ErrCardNotInKitty   = errors.New("card not in kitty")
	ErrMustFollowSuit   = errors.New("must follow suit if able")
	ErrInvalidTrump     = errors.New("invalid trump suit")
	ErrMustDiscardToSix = errors.New("must discard down to 6 cards")
)

// ApplyAction applies an action to the game state and returns the new state
func ApplyAction(state *GameState, action Action) (*GameState, error) {
	switch action.Type {
	case ActionJoinSeat:
		return applyJoinSeat(state, action)
	case ActionLeaveSeat:
		return applyLeaveSeat(state, action)
	case ActionChangeName:
		return applyChangeName(state, action)
	case ActionKickPlayer:
		return applyKickPlayer(state, action)
	case ActionTransferHouse:
		return applyTransferHouse(state, action)
	case ActionStartGame:
		return applyStartGame(state, action)
	case ActionPlaceBid:
		return applyPlaceBid(state, action)
	case ActionSelectTrump:
		return applySelectTrump(state, action)
	case ActionTakeKitty:
		return applyTakeKitty(state, action)
	case ActionDiscard:
		return applyDiscard(state, action)
	case ActionDiscardDraw:
		return applyDiscardDraw(state, action)
	case ActionPlayCard:
		return applyPlayCard(state, action)
	case ActionResetGame:
		return applyResetGame(state, action)
	default:
		return nil, ErrInvalidAction
	}
}

func applyJoinSeat(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseLobby {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex < 0 || action.PlayerIndex > 3 {
		return nil, errors.New("invalid seat index")
	}
	if state.Players[action.PlayerIndex] != nil {
		return nil, ErrSeatTaken
	}

	// Check if this is the first player joining - they become house and dealer
	isFirstPlayer := state.House == -1

	state.Players[action.PlayerIndex] = &Player{
		Name:         action.PlayerName,
		SeatIndex:    action.PlayerIndex,
		SessionToken: GenerateSessionToken(),
		Connected:    true,
	}

	// First player to join becomes the house and dealer
	if isFirstPlayer {
		state.House = action.PlayerIndex
		state.Dealer = action.PlayerIndex
	}

	return state, nil
}

func applyLeaveSeat(state *GameState, action Action) (*GameState, error) {
	if action.PlayerIndex < 0 || action.PlayerIndex > 3 {
		return nil, errors.New("invalid seat index")
	}
	if state.Players[action.PlayerIndex] == nil {
		return nil, ErrSeatEmpty
	}

	// In lobby phase, completely remove the player
	// In other phases, just mark as disconnected so another player can take over
	if state.Phase == PhaseLobby {
		state.Players[action.PlayerIndex] = nil
	} else {
		state.Players[action.PlayerIndex].Connected = false
		state.Players[action.PlayerIndex].Name = "" // Clear name so others know seat is available
	}
	return state, nil
}

func applyStartGame(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseLobby {
		return nil, ErrInvalidAction
	}
	// Only the house can start the game
	if action.PlayerIndex != state.House {
		return nil, errors.New("only the house can start the game")
	}
	if !state.AllPlayersSeated() {
		return nil, ErrNotEnoughPlayers
	}

	// Initialize deck and deal
	state.Deck = NewDeck()
	state.Deck.Shuffle()

	// Deal 6 cards to each player
	for i := 0; i < 4; i++ {
		state.Players[i].Hand = state.Deck.Deal(6)
	}

	// Deal 6 cards to the kitty (center of table)
	state.Kitty = state.Deck.Deal(6)

	// Start bidding with player after dealer
	state.Phase = PhaseBidding
	state.CurrentPlayer = state.PlayerAfterDealer()
	state.Bids = []Bid{}
	state.Trump = nil
	state.TricksPlayed = 0
	state.CompletedTricks = []CompletedTrick{}
	state.CardsWon = [2][]Card{{}, {}}
	state.LastTrick = nil
	state.DiscardComplete = [4]bool{}
	state.PendingDiscards = [4][]string{}
	state.TrumpBroken = false

	return state, nil
}

func applyPlaceBid(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseBidding {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex != state.CurrentPlayer {
		return nil, ErrNotYourTurn
	}

	// Validate bid amount
	// 0 = pass, 2-6 = valid bid (max 5 points: High, Low, Jack, Off Jack, Game)
	// Must bid higher than current high bid (unless passing)
	highBid := 0
	for _, b := range state.Bids {
		if b.Amount > highBid {
			highBid = b.Amount
		}
	}

	if action.BidAmount != 0 {
		if action.BidAmount < 2 || action.BidAmount > 6 {
			return nil, ErrInvalidBid
		}
		if action.BidAmount <= highBid {
			return nil, fmt.Errorf("must bid higher than %d", highBid)
		}
	}

	// Special case: dealer must take high bid if all others pass
	isDealer := action.PlayerIndex == state.Dealer
	allOthersPassed := len(state.Bids) == 3 && highBid == 0

	if isDealer && allOthersPassed && action.BidAmount == 0 {
		// Dealer forced to bid 2
		action.BidAmount = 2
	}

	state.Bids = append(state.Bids, Bid{
		PlayerIndex: action.PlayerIndex,
		Amount:      action.BidAmount,
	})

	// Check if bidding is complete (4 bids placed)
	if len(state.Bids) == 4 {
		// Find winner
		highBid = 0
		bidWinner := state.Dealer // Default to dealer
		for _, b := range state.Bids {
			if b.Amount > highBid {
				highBid = b.Amount
				bidWinner = b.PlayerIndex
			}
		}

		state.BidWinner = bidWinner
		state.WinningBid = highBid
		state.CurrentPlayer = bidWinner
		// Go to kitty phase - bid winner selects trump and picks from kitty
		state.Phase = PhaseKitty
	} else {
		state.CurrentPlayer = NextPlayer(state.CurrentPlayer)
	}

	return state, nil
}

func applySelectTrump(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseKitty {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex != state.BidWinner {
		return nil, ErrNotYourTurn
	}

	// Parse the trump suit
	var trump Suit
	switch action.TrumpSuit {
	case "spades":
		trump = Spades
	case "hearts":
		trump = Hearts
	case "diamonds":
		trump = Diamonds
	case "clubs":
		trump = Clubs
	default:
		return nil, ErrInvalidTrump
	}

	state.Trump = &trump
	return state, nil
}

func applyTakeKitty(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseKitty {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex != state.BidWinner {
		return nil, ErrNotYourTurn
	}

	player := state.Players[action.PlayerIndex]

	// Find and move the specified cards from kitty to player's hand
	for _, cardID := range action.CardIDs {
		cardIdx := -1
		var card Card
		for i, c := range state.Kitty {
			if c.ID == cardID {
				cardIdx = i
				card = c
				break
			}
		}
		if cardIdx == -1 {
			return nil, ErrCardNotInKitty
		}

		// Remove from kitty
		state.Kitty = append(state.Kitty[:cardIdx], state.Kitty[cardIdx+1:]...)
		// Add to hand
		player.Hand = append(player.Hand, card)
	}

	// Clear remaining kitty (they chose not to take these)
	state.Kitty = []Card{}

	return state, nil
}

func applyDiscard(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseKitty {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex != state.BidWinner {
		return nil, ErrNotYourTurn
	}

	// Trump must be selected first
	if state.Trump == nil {
		return nil, errors.New("must select trump before discarding")
	}

	// Kitty must be empty (all cards have been taken or passed on)
	if len(state.Kitty) > 0 {
		return nil, errors.New("must take cards from kitty first")
	}

	player := state.Players[action.PlayerIndex]

	// Discard the specified cards
	for _, cardID := range action.CardIDs {
		cardIdx := -1
		for i, c := range player.Hand {
			if c.ID == cardID {
				cardIdx = i
				break
			}
		}
		if cardIdx == -1 {
			return nil, ErrCardNotInHand
		}

		// Remove from hand
		player.Hand = append(player.Hand[:cardIdx], player.Hand[cardIdx+1:]...)
	}

	// Check if player has too many cards
	if len(player.Hand) > 6 {
		return nil, ErrMustDiscardToSix
	}

	// Deal cards to bid winner if they have fewer than 6
	if len(player.Hand) < 6 && state.Deck != nil {
		needed := 6 - len(player.Hand)
		newCards := state.Deck.Deal(needed)
		player.Hand = append(player.Hand, newCards...)
	}

	// Clear remaining kitty (discarded)
	state.Kitty = []Card{}

	// Transition to discard phase - each player can discard and draw
	// Bid winner is already done (they just discarded and got dealt to 6)
	state.Phase = PhaseDiscard
	state.CurrentPlayer = state.PlayerAfterDealer()
	state.DiscardComplete = [4]bool{}
	state.PendingDiscards = [4][]string{}
	state.DiscardComplete[state.BidWinner] = true // Bid winner already done

	// Skip bid winner if they're first in order
	if state.CurrentPlayer == state.BidWinner {
		processPendingDiscards(state)
	}

	return state, nil
}

// applyDiscardDraw handles a player discarding cards and drawing replacements
// Players can submit their discards at any time during the discard phase,
// but cards are drawn in order (starting after dealer, dealer goes last)
func applyDiscardDraw(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhaseDiscard {
		return nil, ErrInvalidAction
	}
	if state.DiscardComplete[action.PlayerIndex] {
		return nil, errors.New("already completed discard")
	}

	// If it's not this player's turn, store as pending discard
	if action.PlayerIndex != state.CurrentPlayer {
		// Store the pending discard selection
		state.PendingDiscards[action.PlayerIndex] = action.CardIDs
		return state, nil
	}

	// It's this player's turn - process the discard
	err := processDiscard(state, action.PlayerIndex, action.CardIDs)
	if err != nil {
		return nil, err
	}

	// Process any pending discards for subsequent players
	processPendingDiscards(state)

	return state, nil
}

// processDiscard handles the actual discard and draw for a player
func processDiscard(state *GameState, playerIndex int, cardIDs []string) error {
	player := state.Players[playerIndex]

	// Discard the specified cards
	discardCount := len(cardIDs)
	for _, cardID := range cardIDs {
		cardIdx := -1
		for i, c := range player.Hand {
			if c.ID == cardID {
				cardIdx = i
				break
			}
		}
		if cardIdx == -1 {
			return ErrCardNotInHand
		}

		// Remove from hand
		player.Hand = append(player.Hand[:cardIdx], player.Hand[cardIdx+1:]...)
	}

	// Draw replacement cards from deck
	if discardCount > 0 && state.Deck != nil {
		newCards := state.Deck.Deal(discardCount)
		player.Hand = append(player.Hand, newCards...)
	}

	// Mark this player as done and clear pending
	state.DiscardComplete[playerIndex] = true
	state.PendingDiscards[playerIndex] = nil

	return nil
}

// processPendingDiscards processes any pending discards in turn order
func processPendingDiscards(state *GameState) {
	// Move to next player who hasn't discarded yet
	for {
		nextPlayer := NextPlayer(state.CurrentPlayer)
		allDone := true

		for i := 0; i < 4; i++ {
			if !state.DiscardComplete[nextPlayer] {
				allDone = false
				state.CurrentPlayer = nextPlayer

				// Check if this player has pending discards
				if len(state.PendingDiscards[nextPlayer]) > 0 || state.PendingDiscards[nextPlayer] != nil {
					// Process their pending discard (even if empty - means keep all)
					cardIDs := state.PendingDiscards[nextPlayer]
					if cardIDs == nil {
						cardIDs = []string{}
					}
					err := processDiscard(state, nextPlayer, cardIDs)
					if err != nil {
						// If there's an error, clear the pending and let them try again
						state.PendingDiscards[nextPlayer] = nil
						return
					}
					// Continue to check next player
					break
				}
				// No pending discard, wait for this player
				return
			}
			nextPlayer = NextPlayer(nextPlayer)
		}

		if allDone {
			// All players have discarded, start playing
			state.Phase = PhasePlaying
			state.CurrentPlayer = state.BidWinner
			state.CurrentTrick = &Trick{
				Cards:  []TrickCard{},
				Leader: state.BidWinner,
			}
			return
		}
	}
}

func applyPlayCard(state *GameState, action Action) (*GameState, error) {
	if state.Phase != PhasePlaying {
		return nil, ErrInvalidAction
	}
	if action.PlayerIndex != state.CurrentPlayer {
		return nil, ErrNotYourTurn
	}

	player := state.Players[action.PlayerIndex]

	// Find card in hand
	cardIdx := -1
	var playedCard Card
	for i, c := range player.Hand {
		if c.ID == action.CardID {
			cardIdx = i
			playedCard = c
			break
		}
	}
	if cardIdx == -1 {
		return nil, ErrCardNotInHand
	}

	// Trump is already selected during kitty phase
	trump := *state.Trump
	playedIsTrump := playedCard.IsTrump(trump)

	// First card of trick establishes lead suit
	// Off Jack leads trump, not its native suit
	if len(state.CurrentTrick.Cards) == 0 {
		if playedIsTrump {
			state.CurrentTrick.LeadSuit = trump
		} else {
			state.CurrentTrick.LeadSuit = playedCard.Suit
		}
	}

	// Check follow-suit rule
	// Must follow lead suit if able
	// Off Jack counts as trump (not its original suit) for follow-suit purposes
	leadSuit := state.CurrentTrick.LeadSuit
	trumpLed := leadSuit == trump

	// Check if player has the lead suit in their hand
	hasLeadSuit := false
	for _, c := range player.Hand {
		if trumpLed {
			// Trump led: check if player has any trump (including Off Jack)
			if c.IsTrump(trump) {
				hasLeadSuit = true
				break
			}
		} else {
			// Non-trump led: check if player has lead suit (Off Jack doesn't count as its native suit)
			if c.Suit == leadSuit && !c.IsTrump(trump) {
				hasLeadSuit = true
				break
			}
		}
	}

	// If not leading the trick
	if len(state.CurrentTrick.Cards) > 0 {
		// Must follow lead suit if able
		if trumpLed {
			// Trump led: must play trump if you have it
			if !playedIsTrump && hasLeadSuit {
				return nil, ErrMustFollowSuit
			}
		} else {
			// Non-trump led: must follow suit if you have it
			// Can only play trump if you don't have the lead suit
			if hasLeadSuit {
				// Must follow suit - can't play trump or other suits
				if playedCard.Suit != leadSuit && !playedIsTrump {
					return nil, ErrMustFollowSuit
				}
				if playedIsTrump {
					return nil, ErrMustFollowSuit
				}
			}
			// If no lead suit, can play anything (including trump)
		}
	}

	// Play the card
	state.CurrentTrick.Cards = append(state.CurrentTrick.Cards, TrickCard{
		Card:        playedCard,
		PlayerIndex: action.PlayerIndex,
	})

	// Remove card from hand
	player.Hand = append(player.Hand[:cardIdx], player.Hand[cardIdx+1:]...)

	// Check if trick is complete
	if len(state.CurrentTrick.Cards) == 4 {
		// Determine trick winner
		winner := determineTrickWinner(state.CurrentTrick, *state.Trump)
		state.CurrentTrick.Winner = winner
		state.TricksPlayed++

		// Record completed trick
		completedTrick := CompletedTrick{
			Cards:  make([]TrickCard, len(state.CurrentTrick.Cards)),
			Winner: winner,
		}
		copy(completedTrick.Cards, state.CurrentTrick.Cards)
		state.CompletedTricks = append(state.CompletedTricks, completedTrick)

		// Add cards to winning team's pile
		winningTeam := state.GetTeamForPlayer(winner)
		for _, tc := range state.CurrentTrick.Cards {
			state.CardsWon[winningTeam] = append(state.CardsWon[winningTeam], tc.Card)
		}

		// Save last trick for display
		lastTrick := *state.CurrentTrick
		state.LastTrick = &lastTrick

		if state.TricksPlayed == 6 {
			// Hand is complete - score it
			state.Phase = PhaseScoring
			return state, nil
		}

		// Start new trick
		state.CurrentTrick = &Trick{
			Cards:  []TrickCard{},
			Leader: winner,
		}
		state.CurrentPlayer = winner
	} else {
		state.CurrentPlayer = NextPlayer(state.CurrentPlayer)
	}

	return state, nil
}

// determineTrickWinner finds who won the trick
func determineTrickWinner(trick *Trick, trump Suit) int {
	if len(trick.Cards) == 0 {
		return trick.Leader
	}

	winningIdx := 0
	winningCard := trick.Cards[0].Card

	for i := 1; i < len(trick.Cards); i++ {
		card := trick.Cards[i].Card
		if card.Beats(winningCard, trump, trick.LeadSuit) {
			winningIdx = i
			winningCard = card
		}
	}

	return trick.Cards[winningIdx].PlayerIndex
}

// StartNewHand resets for a new hand after scoring
func StartNewHand(state *GameState) *GameState {
	// Rotate dealer
	state.Dealer = NextPlayer(state.Dealer)

	// Reset for new hand
	state.Deck = NewDeck()
	state.Deck.Shuffle()

	for i := 0; i < 4; i++ {
		state.Players[i].Hand = state.Deck.Deal(6)
	}

	// Deal 6 cards to the kitty
	state.Kitty = state.Deck.Deal(6)

	state.Phase = PhaseBidding
	state.CurrentPlayer = state.PlayerAfterDealer()
	state.Bids = []Bid{}
	state.Trump = nil
	state.CurrentTrick = nil
	state.LastTrick = nil
	state.TricksPlayed = 0
	state.CompletedTricks = []CompletedTrick{}
	state.CardsWon = [2][]Card{{}, {}}
	state.BidWinner = -1
	state.WinningBid = 0
	state.DiscardComplete = [4]bool{}
	state.PendingDiscards = [4][]string{}
	state.TrumpBroken = false

	return state
}

// CheckGameOver checks if any team has won
func CheckGameOver(state *GameState) (bool, int) {
	for i, team := range state.Teams {
		if team.Score >= state.TargetScore {
			return true, i
		}
	}
	return false, -1
}

// applyChangeName allows a player to change their name at any time
func applyChangeName(state *GameState, action Action) (*GameState, error) {
	if action.PlayerIndex < 0 || action.PlayerIndex > 3 {
		return nil, errors.New("invalid seat index")
	}
	if state.Players[action.PlayerIndex] == nil {
		return nil, ErrSeatEmpty
	}

	state.Players[action.PlayerIndex].Name = action.PlayerName
	return state, nil
}

// applyTransferHouse transfers house ownership to another player
// Only the current house can do this
func applyTransferHouse(state *GameState, action Action) (*GameState, error) {
	// Only house can transfer
	if action.PlayerIndex != state.House {
		return nil, errors.New("only the house can transfer ownership")
	}

	targetSeat := action.TargetSeat
	if targetSeat < 0 || targetSeat > 3 {
		return nil, errors.New("invalid seat index")
	}

	// Can't transfer to yourself
	if targetSeat == state.House {
		return nil, errors.New("already the house")
	}

	// Target must be a seated, connected player
	if state.Players[targetSeat] == nil || !state.Players[targetSeat].Connected || state.Players[targetSeat].Name == "" {
		return nil, errors.New("target seat is empty or disconnected")
	}

	state.House = targetSeat
	return state, nil
}

// applyKickPlayer kicks a player from their seat
// Only the house can do this
func applyKickPlayer(state *GameState, action Action) (*GameState, error) {
	// Only house can kick
	if action.PlayerIndex != state.House {
		return nil, errors.New("only the house can kick players")
	}

	targetSeat := action.TargetSeat
	if targetSeat < 0 || targetSeat > 3 {
		return nil, errors.New("invalid seat index")
	}

	// Can't kick yourself
	if targetSeat == state.House {
		return nil, errors.New("cannot kick yourself")
	}

	if state.Players[targetSeat] == nil {
		return nil, ErrSeatEmpty
	}

	// In lobby phase, completely remove the player
	// In other phases, mark as disconnected so the seat becomes open
	if state.Phase == PhaseLobby {
		state.Players[targetSeat] = nil
	} else {
		state.Players[targetSeat].Connected = false
		state.Players[targetSeat].Name = ""
		state.Players[targetSeat].SessionToken = "" // Invalidate their session
	}

	return state, nil
}

// applyResetGame resets game to lobby, keeping players seated
// Only the house can do this
func applyResetGame(state *GameState, action Action) (*GameState, error) {
	// Only house can reset
	if action.PlayerIndex != state.House {
		return nil, errors.New("only the house can reset the game")
	}

	// Preserve games won, players, and house
	gamesWon := [2]int{0, 0}
	if state.Teams[0] != nil {
		gamesWon[0] = state.Teams[0].GamesWon
	}
	if state.Teams[1] != nil {
		gamesWon[1] = state.Teams[1].GamesWon
	}

	players := state.Players
	targetScore := state.TargetScore
	house := state.House

	// Reset to fresh game state
	*state = *NewGameState(targetScore)

	// Restore players, games won, and house
	state.Players = players
	state.Teams[0].GamesWon = gamesWon[0]
	state.Teams[1].GamesWon = gamesWon[1]
	state.House = house

	// Clear hands
	for i := 0; i < 4; i++ {
		if state.Players[i] != nil {
			state.Players[i].Hand = nil
		}
	}

	// Set dealer to house
	state.Dealer = house

	return state, nil
}
