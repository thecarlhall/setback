package server

import (
	"errors"
	"log"
	"setback/game"
	"sync"
)

// GameServer handles game logic and message routing
type GameServer struct {
	Hub   *Hub
	State *game.GameState
	mu    sync.Mutex
}

// NewGameServer creates a new game server
func NewGameServer(hub *Hub, targetScore int) *GameServer {
	return &GameServer{
		Hub:   hub,
		State: game.NewGameState(targetScore),
	}
}

// Run starts processing incoming messages
func (gs *GameServer) Run() {
	for msg := range gs.Hub.Incoming {
		gs.HandleMessage(msg.Client, msg.Message)
	}
}

// HandleMessage routes a message to the appropriate handler
func (gs *GameServer) HandleMessage(client *Client, msg ClientMessage) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	var err error

	switch msg.Type {
	case MsgJoinTable:
		err = gs.handleJoinTable(client, msg)
	case MsgLeaveSeat:
		err = gs.handleLeaveSeat(client)
	case MsgChangeName:
		err = gs.handleChangeName(client, msg)
	case MsgKickPlayer:
		err = gs.handleKickPlayer(client, msg)
	case MsgTransferHouse:
		err = gs.handleTransferHouse(client, msg)
	case MsgStartGame:
		err = gs.handleStartGame(client)
	case MsgPlaceBid:
		err = gs.handlePlaceBid(client, msg)
	case MsgSelectTrump:
		err = gs.handleSelectTrump(client, msg)
	case MsgTakeKitty:
		err = gs.handleTakeKitty(client, msg)
	case MsgDiscard:
		err = gs.handleDiscard(client, msg)
	case MsgDiscardDraw:
		err = gs.handleDiscardDraw(client, msg)
	case MsgPlayCard:
		err = gs.handlePlayCard(client, msg)
	case MsgRejoin:
		err = gs.handleRejoin(client, msg)
	case MsgNewHand:
		err = gs.handleNewHand(client)
	case MsgResetGame:
		err = gs.handleResetGame(client)
	default:
		gs.Hub.SendToClient(client, NewErrorMessage("unknown_message", "Unknown message type"))
		return
	}

	if err != nil {
		gs.Hub.SendToClient(client, NewErrorMessage("action_failed", err.Error()))
		return
	}

	// Broadcast state update to all seated players
	gs.broadcastState()
}

func (gs *GameServer) handleJoinTable(client *Client, msg ClientMessage) error {
	if msg.SeatIndex == nil {
		return game.ErrInvalidAction
	}
	seatIndex := *msg.SeatIndex

	// If player is already seated elsewhere, leave that seat first (allows switching seats)
	if client.SeatIndex >= 0 && client.SeatIndex != seatIndex {
		leaveAction := game.Action{
			Type:        game.ActionLeaveSeat,
			PlayerIndex: client.SeatIndex,
		}
		_, err := game.ApplyAction(gs.State, leaveAction)
		if err != nil {
			return err
		}
		gs.Hub.UnseatClient(client)
	}

	// Handle mid-game joining (taking over an empty seat)
	if gs.State.Phase != game.PhaseLobby {
		player := gs.State.Players[seatIndex]
		if player == nil {
			return game.ErrInvalidAction // Can't join new seat mid-game
		}
		if player.Name != "" && player.Connected {
			return game.ErrSeatTaken // Seat is occupied
		}
		// Take over the seat
		player.Name = msg.PlayerName
		player.Connected = true
		player.SessionToken = game.GenerateSessionToken()
		gs.Hub.SeatClient(client, seatIndex)
		client.Token = player.SessionToken
		log.Printf("Player %s took over seat %d mid-game", msg.PlayerName, seatIndex)
		return nil
	}

	action := game.Action{
		Type:        game.ActionJoinSeat,
		PlayerIndex: seatIndex,
		PlayerName:  msg.PlayerName,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	// Seat the client
	gs.Hub.SeatClient(client, seatIndex)
	client.Token = gs.State.Players[seatIndex].SessionToken

	log.Printf("Player %s joined seat %d", msg.PlayerName, seatIndex)
	return nil
}

func (gs *GameServer) handleLeaveSeat(client *Client) error {
	if client.SeatIndex < 0 {
		return game.ErrSeatEmpty
	}

	action := game.Action{
		Type:        game.ActionLeaveSeat,
		PlayerIndex: client.SeatIndex,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	gs.Hub.UnseatClient(client)
	return nil
}

func (gs *GameServer) handleStartGame(client *Client) error {
	action := game.Action{
		Type: game.ActionStartGame,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Game started")
	return nil
}

func (gs *GameServer) handlePlaceBid(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	bidAmount := 0
	if msg.Amount != nil {
		bidAmount = *msg.Amount
	}

	action := game.Action{
		Type:        game.ActionPlaceBid,
		PlayerIndex: client.SeatIndex,
		BidAmount:   bidAmount,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d bid %d", client.SeatIndex, bidAmount)
	return nil
}

func (gs *GameServer) handleSelectTrump(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionSelectTrump,
		PlayerIndex: client.SeatIndex,
		TrumpSuit:   msg.TrumpSuit,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d selected trump: %s", client.SeatIndex, msg.TrumpSuit)
	return nil
}

func (gs *GameServer) handleTakeKitty(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionTakeKitty,
		PlayerIndex: client.SeatIndex,
		CardIDs:     msg.CardIDs,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d took %d cards from kitty", client.SeatIndex, len(msg.CardIDs))
	return nil
}

func (gs *GameServer) handleDiscard(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionDiscard,
		PlayerIndex: client.SeatIndex,
		CardIDs:     msg.CardIDs,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d (bid winner) discarded %d cards, entering discard phase", client.SeatIndex, len(msg.CardIDs))
	return nil
}

func (gs *GameServer) handleDiscardDraw(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionDiscardDraw,
		PlayerIndex: client.SeatIndex,
		CardIDs:     msg.CardIDs,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d discarded %d cards and drew replacements", client.SeatIndex, len(msg.CardIDs))
	return nil
}

func (gs *GameServer) handlePlayCard(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionPlayCard,
		PlayerIndex: client.SeatIndex,
		CardID:      msg.CardID,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player %d played %s", client.SeatIndex, msg.CardID)

	// Check if hand is complete (scoring phase)
	if gs.State.Phase == game.PhaseScoring {
		gs.handleScoring()
	}

	return nil
}

func (gs *GameServer) handleScoring() {
	// Calculate score
	result := game.CalculateScore(gs.State)
	game.ApplyScore(gs.State, result)

	log.Printf("Hand complete. Score: Team 0: %d, Team 1: %d", gs.State.Teams[0].Score, gs.State.Teams[1].Score)

	// Send score update
	scoreMsg := ServerMessage{
		Type:        MsgScoreUpdate,
		ScoreResult: &result,
	}
	gs.Hub.BroadcastMessage(scoreMsg)

	// Check for game over
	if gameOver, winningTeam := game.CheckGameOver(gs.State); gameOver {
		gs.State.Phase = game.PhaseFinished
		gs.State.Teams[winningTeam].GamesWon++
		gameOverMsg := ServerMessage{
			Type:        MsgGameOver,
			WinningTeam: &winningTeam,
		}
		gs.Hub.BroadcastMessage(gameOverMsg)
		log.Printf("Game over! Team %d wins! (Games: %d-%d)", winningTeam,
			gs.State.Teams[0].GamesWon, gs.State.Teams[1].GamesWon)
	}
}

func (gs *GameServer) handleNewHand(client *Client) error {
	if gs.State.Phase != game.PhaseScoring && gs.State.Phase != game.PhaseFinished {
		return game.ErrInvalidAction
	}

	// If game is over, reset to lobby but preserve games won
	if gs.State.Phase == game.PhaseFinished {
		gamesWon := [2]int{gs.State.Teams[0].GamesWon, gs.State.Teams[1].GamesWon}
		gs.State = game.NewGameState(gs.State.TargetScore)
		gs.State.Teams[0].GamesWon = gamesWon[0]
		gs.State.Teams[1].GamesWon = gamesWon[1]
		// Re-add all connected players
		for i := 0; i < 4; i++ {
			if c := gs.Hub.GetClientBySeat(i); c != nil {
				gs.State.Players[i] = &game.Player{
					Name:         "Player " + string(rune('1'+i)),
					SeatIndex:    i,
					SessionToken: c.Token,
					Connected:    true,
				}
			}
		}
		return nil
	}

	// Start new hand
	game.StartNewHand(gs.State)
	log.Printf("New hand started. Dealer: %d", gs.State.Dealer)
	return nil
}

// ErrRejoinFailed is returned when a rejoin attempt fails (stale token)
var ErrRejoinFailed = errors.New("rejoin_failed")

func (gs *GameServer) handleRejoin(client *Client, msg ClientMessage) error {
	// Find player by token
	for i, p := range gs.State.Players {
		if p != nil && p.SessionToken == msg.Token {
			// Rejoin successful
			gs.Hub.SeatClient(client, i)
			client.Token = msg.Token
			p.Connected = true
			log.Printf("Player %s rejoined seat %d", p.Name, i)
			return nil
		}
	}

	return ErrRejoinFailed
}

// broadcastState sends personalized state updates to each player
func (gs *GameServer) broadcastState() {
	// Send to seated players with their hand
	for i := 0; i < 4; i++ {
		if client := gs.Hub.GetClientBySeat(i); client != nil {
			msg := NewStateUpdateMessage(gs.State, i)
			gs.Hub.SendToClient(client, msg)
		}
	}

	// Send to spectators (no hand info)
	for client := range gs.Hub.Clients {
		if client.SeatIndex < 0 {
			msg := NewStateUpdateMessage(gs.State, -1)
			gs.Hub.SendToClient(client, msg)
		}
	}
}

// HandleDisconnect handles a client disconnecting
func (gs *GameServer) HandleDisconnect(client *Client) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if client.SeatIndex >= 0 && client.SeatIndex < 4 {
		if p := gs.State.Players[client.SeatIndex]; p != nil {
			p.Connected = false
			log.Printf("Player %s disconnected from seat %d", p.Name, client.SeatIndex)
		}
	}

	gs.broadcastState()
}

func (gs *GameServer) handleChangeName(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrSeatEmpty
	}

	action := game.Action{
		Type:        game.ActionChangeName,
		PlayerIndex: client.SeatIndex,
		PlayerName:  msg.PlayerName,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Player in seat %d changed name to %s", client.SeatIndex, msg.PlayerName)
	return nil
}

func (gs *GameServer) handleResetGame(client *Client) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	action := game.Action{
		Type:        game.ActionResetGame,
		PlayerIndex: client.SeatIndex,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("Game reset by Player 1 (seat 0)")
	return nil
}

func (gs *GameServer) handleKickPlayer(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	if msg.SeatIndex == nil {
		return game.ErrInvalidAction
	}

	targetSeat := *msg.SeatIndex

	action := game.Action{
		Type:        game.ActionKickPlayer,
		PlayerIndex: client.SeatIndex,
		TargetSeat:  targetSeat,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	// Disconnect the kicked player's client if they're still connected
	if kickedClient := gs.Hub.GetClientBySeat(targetSeat); kickedClient != nil {
		gs.Hub.UnseatClient(kickedClient)
	}

	log.Printf("Player in seat %d was kicked by house", targetSeat)
	return nil
}

func (gs *GameServer) handleTransferHouse(client *Client, msg ClientMessage) error {
	if client.SeatIndex < 0 {
		return game.ErrInvalidAction
	}

	if msg.SeatIndex == nil {
		return game.ErrInvalidAction
	}

	targetSeat := *msg.SeatIndex

	action := game.Action{
		Type:        game.ActionTransferHouse,
		PlayerIndex: client.SeatIndex,
		TargetSeat:  targetSeat,
	}

	_, err := game.ApplyAction(gs.State, action)
	if err != nil {
		return err
	}

	log.Printf("House transferred from seat %d to seat %d", client.SeatIndex, targetSeat)
	gs.broadcastState()
	return nil
}
