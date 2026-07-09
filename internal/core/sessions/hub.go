package sessions

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageType string

const (
	MessageTypeSessionStarted   MessageType = "session:started"
	MessageTypeSessionEnded     MessageType = "session:ended"
	MessageTypeSessionStateSync MessageType = "session:state_sync" // Add for distinguishing between OPEN and STARTED state
)

type Message struct {
	Type MessageType `json:"event_type"`
	Data any         `json:"data,omitempty"`
}

// Logically represents Set(*Client), where Set() is a HashSet
// (every element can be accessed at O(1) time)
type ClientSet map[*Client]struct{}

func NewClientSet() ClientSet {
	return make(ClientSet)
}

func (set ClientSet) AddClient(c *Client) {
	set[c] = struct{}{}
}

func (set ClientSet) RemoveClient(c *Client) {
	delete(set, c)
}

// Represents the state of a user's connection in a session
type ConnState string

const (
	ConnStateConnected    ConnState = "connected"
	ConnStateDisconnected ConnState = "disconnected"
)

type Client struct {
	conn      *websocket.Conn
	send      chan Message
	sessionID uuid.UUID
	userID    uuid.UUID
}

// For tracking user states in sessions (examiners). Useful
// for handling disconnects appropriately
type MemberState struct {
	userID         uuid.UUID
	sessionID      uuid.UUID
	connState      ConnState
	disconnectedAt *time.Time
	graceTimer     *time.Timer
}

func memberKey(userID, sessionID uuid.UUID) string {
	return userID.String() + ":" + sessionID.String()
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]ClientSet // sessionID 		-> Set(*Client) (Basically rooms)
	members map[string]*MemberState // userID:sessionID -> *MemberState (Tracking every client's connection state)
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]ClientSet),
		members: make(map[string]*MemberState),
	}
}

// Adds client to the session. Will return if the registration was
// a reconnected, and how long the client was disconnected for.
func (h *Hub) Register(client *Client) (wasReconnect bool, disconnectedFor time.Duration) {

	h.mu.Lock()
	defer h.mu.Unlock()

	// Evict existing clients with the same userID
	for existing := range h.clients[client.sessionID] {
		if existing.userID == client.userID {
			close(existing.send)
			h.clients[client.sessionID].RemoveClient(existing)
		}
	}

	// Session isn't tracked yet, create new ClientSet and add
	// the client to the set of clients in the session.

	if h.clients[client.sessionID] == nil {
		h.clients[client.sessionID] = NewClientSet()
	}
	h.clients[client.sessionID].AddClient(client)

	// Check if state of the client is tracked i.e. the client
	// is reconnecting.
	//
	// If they don't have a tracked state, client is connecting
	// for the first time in that session, create a new state for
	// them and return

	key := memberKey(client.userID, client.sessionID)
	member, exists := h.members[key]

	if !exists {
		h.members[key] = &MemberState{
			userID:    client.userID,
			sessionID: client.sessionID,
			connState: ConnStateConnected,
		}
		return false, 0
	}

	// Cancel grace timer if client reconnects within the
	// grace period (Should trigger automically from time.AfterFunc if otherwise)

	if member.graceTimer != nil {
		member.graceTimer.Stop()
		member.graceTimer = nil
	}

	if member.disconnectedAt != nil {
		disconnectedFor = time.Since(*member.disconnectedAt)
		wasReconnect = true
	}

	member.connState = ConnStateConnected
	member.disconnectedAt = nil
	return
}

// Removes client from the session
func (h *Hub) Unregister(client *Client, onGraceExpired func()) {

	h.mu.Lock()

	// Remove the client from the session, delete the session's
	// client set if that was the last client.

	h.clients[client.sessionID].RemoveClient(client)
	if len(h.clients[client.sessionID]) == 0 {
		delete(h.clients, client.sessionID)
	}

	// !ok should not be possible in theory but it is just a guard
	// to avoid panics.

	key := memberKey(client.userID, client.sessionID)
	member, ok := h.members[key]
	if !ok {
		h.mu.Unlock()
		return
	}

	// Sets disconnectedAt and connState as disconnected.

	now := time.Now()
	member.connState = ConnStateDisconnected
	member.disconnectedAt = &now

	h.mu.Unlock() // release before AfterFunc to avoid holding lock in timer

	// Will run onGraceExpired if the client does not call Register()
	// after the grace period.

	gracePeriod := 40 * time.Second
	member.graceTimer = time.AfterFunc(gracePeriod, onGraceExpired)
}

// Broadcast message to all clients connected to session
func (h *Hub) Broadcast(sessionID uuid.UUID, msg Message) {

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[sessionID] { // Look through all clients to send message
		select {
		case client.send <- msg:
			log.Printf(
				"Session:(%s) Sent to client message type \"%s\"\n",
				sessionID,
				msg.Type,
			)

		default: // Ignores client immediately if client.send is blocked
		}
	}
}

// Send a message to a particular user within a session
func (h *Hub) SendToUser(sessionID, userID uuid.UUID, msg Message) {

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[sessionID] {
		if client.userID == userID { // Found user
			select {
			case client.send <- msg:
			default:
				go client.conn.Close()
			}
			return
		}
	}
}

// Close the session, disconnecting all connected clients
func (h *Hub) CloseSession(sessionID uuid.UUID) {

	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients[sessionID] {
		close(client.send)
	}
}
