package server

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	SeatIndex int // -1 if not seated
	Token     string
}

// Hub manages all WebSocket connections and game state
type Hub struct {
	Clients    map[*Client]bool
	Seats      [4]*Client // Clients by seat index
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	Incoming   chan *ClientMessageWithSender
	mu         sync.RWMutex
}

// ClientMessageWithSender pairs a message with its sender
type ClientMessageWithSender struct {
	Client  *Client
	Message ClientMessage
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Seats:      [4]*Client{},
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Incoming:   make(chan *ClientMessageWithSender, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				if client.SeatIndex >= 0 && client.SeatIndex < 4 {
					h.Seats[client.SeatIndex] = nil
				}
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(client *Client, msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	select {
	case client.Send <- data:
	default:
		log.Printf("Client send buffer full")
	}
}

// SendToSeat sends a message to the client at a specific seat
func (h *Hub) SendToSeat(seatIndex int, msg ServerMessage) {
	h.mu.RLock()
	client := h.Seats[seatIndex]
	h.mu.RUnlock()

	if client != nil {
		h.SendToClient(client, msg)
	}
}

// BroadcastMessage sends a message to all clients
func (h *Hub) BroadcastMessage(msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling broadcast: %v", err)
		return
	}
	h.Broadcast <- data
}

// SeatClient assigns a client to a seat
func (h *Hub) SeatClient(client *Client, seatIndex int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from old seat if any
	if client.SeatIndex >= 0 && client.SeatIndex < 4 {
		h.Seats[client.SeatIndex] = nil
	}

	client.SeatIndex = seatIndex
	if seatIndex >= 0 && seatIndex < 4 {
		h.Seats[seatIndex] = client
	}
}

// UnseatClient removes a client from their seat
func (h *Hub) UnseatClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.SeatIndex >= 0 && client.SeatIndex < 4 {
		h.Seats[client.SeatIndex] = nil
	}
	client.SeatIndex = -1
}

// GetClientBySeat returns the client at a seat
func (h *Hub) GetClientBySeat(seatIndex int) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if seatIndex >= 0 && seatIndex < 4 {
		return h.Seats[seatIndex]
	}
	return nil
}

// GetClientByToken finds a client by session token
func (h *Hub) GetClientByToken(token string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		if client.Token == token {
			return client
		}
	}
	return nil
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		c.Hub.Incoming <- &ClientMessageWithSender{
			Client:  c,
			Message: clientMsg,
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}
