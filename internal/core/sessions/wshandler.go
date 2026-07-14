package sessions

import (
	"context"
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
// handled inside the route itself. Routes include:
//
// - /sessions/{session_id}?token
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

	// -------------------------------------------------------------
	// --- Access Guards -------------------------------------------
	// -------------------------------------------------------------

	session, err := h.svc.FindSessionByID(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	if session.Status != store.SessionStatusOpen && session.Status != store.SessionStatusStarted {
		json.WriteJSON(w, http.StatusForbidden, "Session is not currently accessible", nil)
		return
	}

	canConnect, err := h.svc.ExamineeCanConnect(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, "Cannot verify examinee state", nil)
		return
	}
	if !canConnect {
		json.WriteJSON(w, http.StatusForbidden, "Cannot connect to session", nil)
		return
	}

	// -------------------------------------------------------------
	// --- Upgrading to Websocket Connection -----------------------
	// -------------------------------------------------------------

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

	// Attempt to set the client's disconnection state
	// then unregister the client from the Hub.

	defer func() {

		ctx := context.Background() // Request ctx is dead by now

		if err := h.svc.OnExamineeDisconnect(ctx, user, sessionID); err != nil {
			log.Printf(
				"OnExamineeDisconnect sessionID:(%s) examineeID:(%s): %v",
				sessionID, user.ID, err,
			)
		}

		var onGraceExpired func()
		if status, err := h.svc.FindSubmissionStatus(
			ctx, user, sessionID,
		); err == nil && status == store.SubmissionStatusDisconnected {
			onGraceExpired = func() {
				if err := h.svc.OnGraceExpired(ctx, user.ID, client.sessionID); err != nil {
					log.Printf(
						"OnGraceExpired sessionID:(%s) examineeID:(%s): %v",
						sessionID, user.ID, err,
					)
				}
			}
		}

		h.hub.Unregister(client, onGraceExpired)
	}()

	// Set the client's submission status and then register
	// the client into the Hub

	if err := h.svc.OnExamineeConnect(r.Context(), user, sessionID); err != nil {
		conn.Close()
		return
	}
	wasReconnect, disconnectedFor := h.hub.Register(client)

	if wasReconnect {
		log.Printf(
			"Session:(%s) User reconnected after %.fs",
			client.sessionID,
			disconnectedFor.Seconds(),
		)

		h.svc.SendStateSync(r.Context(), client.userID, client.sessionID)
	}

	go h.writePump(client)
	h.readPump(r.Context(), user, client)
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
func (h *WebsocketHandler) writePump(client *Client) {

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
func (h *WebsocketHandler) readPump(ctx context.Context, user store.User, client *Client) {

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
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		log.Printf(
			"Session:(%s) Receieved data from client (%s)\n",
			client.sessionID,
			client.userID,
		)

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to process message\n")
			continue // Just ignore it and continue
		}

		payload, err := json.Marshal(msg.Data)
		if err != nil {
			log.Printf("Failed to process message data\n")
			continue // Just ignore it and continue
		}

		switch msg.Type {
		case MessageTypeProctorEvent:
			log.Printf(
				"Session:(%s) Proctor message received from client (%s), type: %s\n",
				client.sessionID,
				client.userID,
				msg.Type,
			)

			var event ProctorEventData
			if err := json.Unmarshal(payload, &event); err != nil {
				log.Printf(
					"Session:(%s) Failed to process proctor event received from client (%s)\n",
					client.sessionID,
					client.userID,
				)
				continue
			}

			event.SessionID = client.sessionID
			event.ExamineeID = client.userID

			if err := h.svc.HandleProctorEvent(ctx, user, event); err != nil {
				log.Printf(
					"Session:(%s) Failed to handle proctor event received from client (%s)\n",
					client.sessionID,
					client.userID,
				)
				continue
			}

		}
	}

	log.Printf(
		"Session:(%s) User unreachable for now...\n",
		client.sessionID,
	)
}
