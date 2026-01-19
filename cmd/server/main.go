package main

import (
	"flag"
	"log"
	"net/http"
	"setback/server"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

func main() {
	port := flag.String("port", "8080", "Server port")
	targetScore := flag.Int("target", 52, "Target score to win")
	flag.Parse()

	// Create hub and game server
	hub := server.NewHub()
	gameServer := server.NewGameServer(hub, *targetScore)

	// Start hub and game server in background
	go hub.Run()
	go gameServer.Run()

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := &server.Client{
			Hub:       hub,
			Conn:      conn,
			Send:      make(chan []byte, 256),
			SeatIndex: -1,
		}

		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()

		// Send initial state
		hub.SendToClient(client, server.NewStateUpdateMessage(gameServer.State, -1))
	})

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	addr := ":" + *port
	log.Printf("Starting Setback server on http://localhost%s", addr)
	log.Printf("Target score: %d", *targetScore)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
