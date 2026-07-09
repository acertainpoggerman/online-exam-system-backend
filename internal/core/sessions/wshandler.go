package sessions

import (
	"log"
	"net/http"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Tighten to prevent CSRF
	},
}

type WebsocketHandler struct {
	svc          SessionService
	hub          *Hub
	jwtSecretKey []byte
}

func NewWebsocketHandler(
	svc SessionService,
	hub *Hub,
	jwtSecretKey []byte,
) *WebsocketHandler {
	return &WebsocketHandler{svc, hub, jwtSecretKey}
}

// Register under unauthenticated route, Authentication will be
// handled inside the route itself.
func (h *WebsocketHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("/sessions/{session_id}", h.HandleSessionConnect)
}

//------------------------------------------------------------------------------------
// --- Routes ------------------------------------------------------------------------
//------------------------------------------------------------------------------------

func (h *WebsocketHandler) HandleSessionConnect(w http.ResponseWriter, r *http.Request) {

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	userPtr, err := jwt.ValidateJWT(token, h.jwtSecretKey)
	if err != nil || userPtr == nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	user := *userPtr

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	session, err := h.svc.FindSessionByID(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	if session.Status != store.SessionStatusOpen && session.Status != store.SessionStatusStarted {
		json.WriteJSON(w, http.StatusForbidden, "Session is not currently accessible", nil)
		return
	}

	inSession, err := h.svc.ExamineeInSession(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, "Failed to verify enrollment", nil)
		return
	}
	if !inSession {
		json.WriteJSON(w, http.StatusForbidden, "Not enrolled in this session", nil)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		conn:      conn,
		send:      make(chan Message, 8),
		sessionID: sessionID,
		userID:    user.ID,
	}

	wasReconnect, disconnectedFor := h.hub.Register(client)
	defer h.hub.Unregister(client, func() {
		h.svc.OnGraceExpired(user.ID, client.sessionID)
	})

	if wasReconnect {
		log.Printf(
			"Session:(%s) User reconnected after %.fs",
			client.sessionID,
			disconnectedFor.Seconds(),
		)
		h.svc.SendStateSync(r.Context(), client.userID, client.sessionID)
	}

	go writePump(client)
	readPump(client)
}

//------------------------------------------------------------------------------------
// --- Websocket Pumps ---------------------------------------------------------------
//------------------------------------------------------------------------------------

// Handles writing to the connection:
//
// A ping sent every 25 seconds, if there is no corresponding pong received, client
// has disconnected, and the connection is closed
//
// Also waits for a message to write to the connection, closes
// connection if channel has been closed, else writes the message
// to the connection.
func writePump(client *Client) {

	defer client.conn.Close()
	pinger := time.Tick(15 * time.Second) // For continuous pinging

	// The amount of time within conn.SetWriteDeadline() being called
	// that conn.Write must execute, or else the conn is considered dead
	deadlineDuration := 5 * time.Second

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(deadlineDuration))

			// Channel has been closed and there are no more messages to receive,
			// write close message and trigger conn.Close()
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(msg); err != nil {
				return
			}

		// Pings are sent to ensure that the readPumps Pong's deadline
		// is not reached i.e. the client will respond with a Pong message
		// which resets the deadline.
		case <-pinger:
			client.conn.SetWriteDeadline(time.Now().Add(deadlineDuration))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Handles reading from the connection:
//
// A pong handler is set to extend the read deadline if it successfully
// responds to a ping to the connection.
//
// conn.ReadMessage will handling proctor events.
func readPump(client *Client) {

	defer client.conn.Close()

	// The amount of time the client has to make a read
	// before its connection is considered dead. Used for checking
	// if the client has disconnected here.
	deadlineDuration := 20 * time.Second

	client.conn.SetReadDeadline(time.Now().Add(deadlineDuration))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(deadlineDuration))
		return nil
	})

	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			break
		}
	}

	log.Printf(
		"Session:(%s) User unreachable for now...\n",
		client.sessionID,
	)
}
