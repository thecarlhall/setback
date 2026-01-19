package game

// ScoreResult contains the scoring breakdown for a hand
type ScoreResult struct {
	HighTeam     int    `json:"highTeam"`     // Team that gets High point (-1 if no trump played)
	HighCard     string `json:"highCard"`     // The high trump card
	LowTeam      int    `json:"lowTeam"`      // Team that gets Low point (-1 if no trump played)
	LowCard      string `json:"lowCard"`      // The low trump card
	JackTeam     int    `json:"jackTeam"`     // Team that captured Jack of trump (-1 if not played)
	OffJackTeam  int    `json:"offJackTeam"`  // Team that captured Off Jack (-1 if not played)
	GameTeam     int    `json:"gameTeam"`     // Team with most game points (-1 if tie)
	Team0Points  int    `json:"team0Points"`  // Total points for team 0
	Team1Points  int    `json:"team1Points"`  // Total points for team 1
	BidderTeam   int    `json:"bidderTeam"`   // Which team bid
	BidAmount    int    `json:"bidAmount"`    // The winning bid
	BidMade      bool   `json:"bidMade"`      // Did bidding team make their bid?
	Team0Change  int    `json:"team0Change"`  // Score change for team 0
	Team1Change  int    `json:"team1Change"`  // Score change for team 1
	GamePoints   [2]int `json:"gamePoints"`   // Game point totals per team
}

// CalculateScore scores a completed hand
// See: https://www.singaporemahjong.com/pitch/rules/
func CalculateScore(state *GameState) ScoreResult {
	result := ScoreResult{
		HighTeam:    -1,
		LowTeam:     -1,
		JackTeam:    -1,
		OffJackTeam: -1,
		GameTeam:    -1,
		BidderTeam:  state.GetTeamForPlayer(state.BidWinner),
		BidAmount:   state.WinningBid,
	}

	if state.Trump == nil {
		return result
	}

	trump := *state.Trump

	// Find High and Low trump from completed tricks
	// High/Low go to the team that PLAYED them (not captured)
	// Note: Off Jack counts as trump for play but not for High/Low
	var highCard *Card
	var lowCard *Card
	var highPlayer, lowPlayer int

	for _, trick := range state.CompletedTricks {
		for _, tc := range trick.Cards {
			// Only actual trump suit cards count for High/Low (not Off Jack)
			if tc.Card.Suit == trump {
				if highCard == nil || tc.Card.Rank > highCard.Rank {
					card := tc.Card
					highCard = &card
					highPlayer = tc.PlayerIndex
				}
				if lowCard == nil || tc.Card.Rank < lowCard.Rank {
					card := tc.Card
					lowCard = &card
					lowPlayer = tc.PlayerIndex
				}
			}
		}
	}

	if highCard != nil {
		result.HighTeam = state.GetTeamForPlayer(highPlayer)
		result.HighCard = highCard.ID
	}
	if lowCard != nil {
		result.LowTeam = state.GetTeamForPlayer(lowPlayer)
		result.LowCard = lowCard.ID
	}

	// Find Jack of trump - goes to the team that CAPTURED it (won the trick)
	for _, trick := range state.CompletedTricks {
		for _, tc := range trick.Cards {
			if tc.Card.Suit == trump && tc.Card.Rank == Jack {
				result.JackTeam = state.GetTeamForPlayer(trick.Winner)
				break
			}
		}
		if result.JackTeam >= 0 {
			break
		}
	}

	// Find Off Jack - goes to the team that CAPTURED it (won the trick)
	// Off Jack is the Jack of the same color suit
	offSuit := trump.OffSuit()
	for _, trick := range state.CompletedTricks {
		for _, tc := range trick.Cards {
			if tc.Card.Suit == offSuit && tc.Card.Rank == Jack {
				result.OffJackTeam = state.GetTeamForPlayer(trick.Winner)
				break
			}
		}
		if result.OffJackTeam >= 0 {
			break
		}
	}

	// Calculate Game points from cards won by each team
	// A=4, K=3, Q=2, J=1, 10=10
	gamePoints := [2]int{0, 0}
	for team := 0; team < 2; team++ {
		for _, card := range state.CardsWon[team] {
			gamePoints[team] += card.Rank.GamePoints()
		}
	}

	result.GamePoints = gamePoints
	if gamePoints[0] > gamePoints[1] {
		result.GameTeam = 0
	} else if gamePoints[1] > gamePoints[0] {
		result.GameTeam = 1
	}
	// If tie, neither team gets Game point

	// Calculate total points for each team
	if result.HighTeam == 0 {
		result.Team0Points++
	} else if result.HighTeam == 1 {
		result.Team1Points++
	}

	if result.LowTeam == 0 {
		result.Team0Points++
	} else if result.LowTeam == 1 {
		result.Team1Points++
	}

	if result.JackTeam == 0 {
		result.Team0Points++
	} else if result.JackTeam == 1 {
		result.Team1Points++
	}

	if result.OffJackTeam == 0 {
		result.Team0Points++
	} else if result.OffJackTeam == 1 {
		result.Team1Points++
	}

	if result.GameTeam == 0 {
		result.Team0Points++
	} else if result.GameTeam == 1 {
		result.Team1Points++
	}

	// Apply setback rule
	bidderTeamPoints := result.Team0Points
	if result.BidderTeam == 1 {
		bidderTeamPoints = result.Team1Points
	}

	result.BidMade = bidderTeamPoints >= result.BidAmount

	// Calculate score changes
	if result.BidMade {
		result.Team0Change = result.Team0Points
		result.Team1Change = result.Team1Points
	} else {
		// Bidding team gets set back (loses bid amount)
		if result.BidderTeam == 0 {
			result.Team0Change = -result.BidAmount
			result.Team1Change = result.Team1Points
		} else {
			result.Team0Change = result.Team0Points
			result.Team1Change = -result.BidAmount
		}
	}

	return result
}

// ApplyScore applies the score result to the game state
func ApplyScore(state *GameState, result ScoreResult) {
	state.Teams[0].Score += result.Team0Change
	state.Teams[1].Score += result.Team1Change
}
