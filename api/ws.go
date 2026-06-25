package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ─── Hub: manages connected WebSocket clients ───────────────────────────────

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

var hub = &Hub{
	clients: make(map[*websocket.Conn]bool),
}

func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	log.Printf("ws: client connected (%d total)\n", len(h.clients))
}

func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
	log.Printf("ws: client disconnected (%d total)\n", len(h.clients))
}

// Broadcast sends a JSON message to all connected clients.
func (h *Hub) Broadcast(v interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteJSON(v); err != nil {
			log.Println("ws: write error:", err)
			go h.unregister(conn)
		}
	}
}

// ─── WebSocket upgrade & handler ─────────────────────────────────────────────

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWS upgrades the connection and keeps it alive.
// The client receives broadcasts of training progress and metrics.
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws: upgrade error:", err)
		return
	}

	hub.register(conn)

	// Read loop — keeps the connection open and detects disconnects.
	go func() {
		defer hub.unregister(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// ─── Event types sent over WebSocket ─────────────────────────────────────────

type WSEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type TrainStartEvent struct {
	TotalTrees int `json:"total_trees"`
	NumNodes   int `json:"num_nodes"`
}

type NodeDoneEvent struct {
	NodeID string `json:"node_id"`
	Trees  int    `json:"trees"`
	DurMS  int64  `json:"dur_ms"`
}

type TrainDoneEvent struct {
	TotalTrees int     `json:"total_trees"`
	DurTotalMS int64   `json:"dur_total_ms"`
	MAE        float64 `json:"mae"`
	RMSE       float64 `json:"rmse"`
	R2         float64 `json:"r2"`
}
