package websocket

import (
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // tighten in production
	},
}

type WebsocketHandler struct {
	hub          *Hub
	jwtSecretKey []byte
}

func NewHandler(hub *Hub, jwtSecretKey []byte) *WebsocketHandler {
	return &WebsocketHandler{hub, jwtSecretKey}
}

func (h *WebsocketHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("/sessions/{session_id}", h.HandleSessionConn)
}

//------------------------------------------------------------------------------------
// --- Routes ------------------------------------------------------------------------
//------------------------------------------------------------------------------------

func (h *WebsocketHandler) HandleSessionConn(w http.ResponseWriter, r *http.Request) {

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userData, err := jwt.ValidateJWT(token, h.jwtSecretKey)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		conn:      conn,
		send:      make(chan Message, 8),
		sessionID: sessionID,
		userID:    userData.ID,
	}

	h.hub.Register(sessionID, client)
	defer h.hub.Unregister(sessionID, client)

	// write pump — send messages to client
	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}()

	// read pump — keep connection alive, detect disconnect
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
