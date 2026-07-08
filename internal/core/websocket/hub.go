package websocket

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageType string

const (
	MessageTypeSessionStarted MessageType = "session:started"
	MessageTypeSessionEnded   MessageType = "session:ended"
)

type Message struct {
	Type MessageType `json:"event_type"`
	Data any         `json:"data,omitempty"`
}

type Client struct {
	conn      *websocket.Conn
	send      chan Message
	sessionID uuid.UUID
	userID    uuid.UUID
}

type Hub struct {
	mu sync.RWMutex
	// Map of sessionIDs to a set of clients
	clients map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]struct{}),
	}
}

func (h *Hub) Register(sessionID uuid.UUID, client *Client) {

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[sessionID] == nil {
		h.clients[sessionID] = make(map[*Client]struct{})
	}
	h.clients[sessionID][client] = struct{}{}
}

func (h *Hub) Unregister(sessionID uuid.UUID, client *Client) {

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients[sessionID], client)
	if len(h.clients[sessionID]) == 0 {
		delete(h.clients, sessionID)
	}
}

func (h *Hub) Broadcast(sessionID uuid.UUID, msg Message) {

	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[sessionID] {
		select {
		case client.send <- msg:
		default:
			close(client.send)
		}
	}
}
