package sessions

import (
	"log"
	"sync"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageType string

const (
	// Event indicates that the session has started
	MessageTypeSessionStarted MessageType = "session:started"

	// Event indicates that the session has ended
	MessageTypeSessionEnded MessageType = "session:ended"

	// Event sent to a user who is reconnnecting to the session.
	// useful for signalling that the session is in a STARTED state if
	// the user left while the session was in OPEN state
	MessageTypeSessionStateSync MessageType = "session:state_sync"

	// Event for signalling the user (examinee) has been flagged
	// (could be for extended disconnections or cheating)
	MessageTypeSessionFlagged MessageType = "session:flagged"

	// Event for signalling the user (examinee) has been allowed
	// by the examiner to continue their examiner after being flagged
	MessageTypeSessionUnflagged MessageType = "session:unflagged"

	// Event from clients signaling unusual behaviour and actions
	MessageTypeProctorEvent MessageType = "session:proctor_event"

	MessageTypeParticipantsChanged MessageType = "session:participants_changed"
)

type Message struct {
	Type MessageType `json:"event_type"`
	Data any         `json:"data,omitempty"`
}

type ProctorEventData struct{ store.ProctorEvent }

// ---------------------------------------------------------------
// --- ClientSet -------------------------------------------------
// ---------------------------------------------------------------

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

// ---------------------------------------------------------------
// --- ConnectionState -------------------------------------------
// ---------------------------------------------------------------

// Represents the state of a user's connection in a session
type ConnState string

const (
	ConnStateNone         ConnState = "none"
	ConnStateConnected    ConnState = "connected"
	ConnStateDisconnected ConnState = "disconnected"
)

// ---------------------------------------------------------------
// --- Client ----------------------------------------------------
// ---------------------------------------------------------------

type Client struct {
	conn      *websocket.Conn
	send      chan Message
	sessionID uuid.UUID
	userID    uuid.UUID
	closeSend sync.Once
	role      store.UserRole
}

// Shuts down the client's send channel. Only does it
// once no matter how many times it is called
func (c *Client) shutdownSend() {
	c.closeSend.Do(func() { close(c.send) })
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

// Takes a User ID & Session ID, converting it to 'userID:sessionID'
func memberKey(userID, sessionID uuid.UUID) string {
	return userID.String() + ":" + sessionID.String()
}

// ---------------------------------------------------------------
// --- Hub -------------------------------------------------------
// ---------------------------------------------------------------

// Hub represents a websocket mapping of session rooms and clients connected to each room.
// The hub can be used to:
//
// - Register & unregister client connections
//
// - Broadcast messages to all or specific groups of clients (by roles)
//
// - Send messages to specific clients
//
// The hub also tracks every client's connection state for tracking
// client connection states & for fault tolerance against disconnection
// of clients
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
			existing.shutdownSend()
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

	// Will run onGraceExpired if the client does not call Register()
	// after the grace period.

	gracePeriod := 2 * time.Minute
	if onGraceExpired != nil {
		member.graceTimer = time.AfterFunc(gracePeriod, onGraceExpired)
	}

	h.mu.Unlock()
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

func (h *Hub) BroadcastTo(sessionID uuid.UUID, msg Message, role store.UserRole) {

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[sessionID] {
		if client.role != role {
			continue
		}
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

func (h *Hub) PurgeSession(sessionID uuid.UUID) {

	h.mu.Lock()

	// Clone the set of clients to avoid deadlock
	clients := make([]*Client, 0, len(h.clients[sessionID]))
	for client := range h.clients[sessionID] {
		clients = append(clients, client)
	}

	h.mu.Unlock()

	for _, client := range clients {
		client.shutdownSend()
	}
}

func (h *Hub) RemoveMember(userID, sessionID uuid.UUID) {

	h.mu.Lock()
	defer h.mu.Unlock()

	key := memberKey(userID, sessionID)
	if member, ok := h.members[key]; ok && member.graceTimer != nil {
		member.graceTimer.Stop()
	}
	delete(h.members, key)

	for client := range h.clients[sessionID] {
		if client.userID == userID {
			client.shutdownSend()
		}
	}
}

func (h *Hub) GetMemberConnState(userID, sessionID uuid.UUID) (connState ConnState) {

	h.mu.RLock()
	defer h.mu.RUnlock()

	member, hasState := h.members[memberKey(userID, sessionID)]
	if hasState {
		connState = member.connState
		return
	}

	return ConnStateNone
}
