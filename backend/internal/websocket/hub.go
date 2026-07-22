package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/task-underground/backend/internal/metrics"
)

type Hub struct {
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	redis      *redis.Client // nil → single-instance, in-memory only
}

type Client struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			metrics.WSConnectionsActive.Inc()
			log.Printf("Client connected: %s (user: %s)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				metrics.WSConnectionsActive.Dec()
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.ID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToUser(userID uuid.UUID, message Message) {
	h.send([]uuid.UUID{userID}, message)
}

func (h *Hub) BroadcastToTask(taskID uuid.UUID, message Message, userIDs []uuid.UUID) {
	h.send(userIDs, message)
}

// send routes a message to the given users. With Redis configured it goes out
// over Pub/Sub and comes back through this instance's own subscription, so
// clients on every instance are reached exactly once.
func (h *Hub) send(userIDs []uuid.UUID, message Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	if h.publish(userIDs, data) {
		return
	}
	h.deliver(userIDs, data)
}

// deliver writes data to the locally connected clients belonging to userIDs.
func (h *Hub) deliver(userIDs []uuid.UUID, data []byte) {
	recipients := make(map[uuid.UUID]bool, len(userIDs))
	for _, id := range userIDs {
		recipients[id] = true
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		if !recipients[client.UserID] {
			continue
		}
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.clients, client.ID)
			metrics.WSConnectionsActive.Dec()
		}
	}
}
