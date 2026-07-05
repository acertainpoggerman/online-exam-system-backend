package sessions

import (
	"errors"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type SessionHandler struct {
	svc      SessionService
	upgrader websocket.Upgrader
}

func NewSessionHandler(svc SessionService) *SessionHandler {
	return &SessionHandler{svc, websocket.Upgrader{
		Subprotocols: []string{"base64.binary.oes"},
		CheckOrigin:  func(r *http.Request) bool { return true },
	}}
}

func (h *SessionHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("POST /sessions", h.postSession)
	r.HandleFunc("GET /sessions", h.getSessions)
	r.HandleFunc("GET /sessions/{session_id}", h.getSessionByID)
	r.HandleFunc("PUT /sessions/{session_id}", h.putSessionByID)

	r.HandleFunc("POST /sessions/{session_id}/open", h.openSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/close", h.closeSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/start", h.startSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/end", h.endSessionByID)
}

func (h *SessionHandler) RegisterWebsocketRoutes(r *http.ServeMux) {
	// r.HandleFunc("/sessions/{session_id}", h.joinSessionByID)
}

// -----------------------------------------------------------------------
// --- In-Session Related ------------------------------------------------
// -----------------------------------------------------------------------

// func (h *SessionHandler) joinSessionByID(w http.ResponseWriter, r *http.Request) {

// 	user, err := jwt.GetUserDataFromContext(r.Context())
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
// 		return
// 	}

// 	sessionID, err := uuid.Parse(r.PathValue("session_id"))
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
// 		return
// 	}

// 	if _, err := h.svc.FindSessionByID(r.Context(), user, sessionID); err != nil {
// 		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
// 		return
// 	}

// 	// Websocket part of things

// 	conn, err := h.upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
// 		return
// 	}
// 	defer conn.Close()

// 	if err := h.svc.JoinSessionByID(r.Context(), user, sessionID, conn); err != nil {
// 		log.Printf("Error. Status Code: %d, Reason: %v", http.StatusInternalServerError, err)
// 		return
// 	}
// }

func (h *SessionHandler) openSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
	}

	if err := h.svc.OpenSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) closeSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
	}

	if err := h.svc.CloseSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) startSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
	}

	if err := h.svc.StartSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) endSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
	}

	if err := h.svc.EndSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// -----------------------------------------------------------------------
// --- Basic CRUD Handlers -----------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) postSession(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	var body CreateSessionBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	id, err := h.svc.CreateSession(r.Context(), user, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"session_id": id}, nil)
}

func (h *SessionHandler) getSessions(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessions, err := h.svc.FindSessions(r.Context(), user)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"sessions": sessions}, nil)
}

func (h *SessionHandler) getSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": errors.New("ID passed not valid")}, nil)
		return
	}

	session, err := h.svc.FindSessionByID(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"session": session}, nil)
}

func (h *SessionHandler) putSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": errors.New("ID passed not valid")}, nil)
		return
	}

	var body CreateSessionBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	session, err := h.svc.UpdateSessionByID(r.Context(), user, sessionID, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"session": session}, nil)
}
