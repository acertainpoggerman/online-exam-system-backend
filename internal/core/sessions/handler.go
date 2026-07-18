package sessions

import (
	"errors"
	"net/http"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
)

type SessionHandler struct {
	svc SessionService
}

func NewSessionHandler(svc SessionService) *SessionHandler {
	return &SessionHandler{svc}
}

func (h *SessionHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("POST /sessions", h.postSession)
	r.HandleFunc("GET /sessions", h.getSessions)
	r.HandleFunc("GET /sessions/{session_id}", h.getSessionByID)
	r.HandleFunc("PUT /sessions/{session_id}", h.putSessionByID)

	r.HandleFunc("POST /sessions/join", h.enrolInSession)

	r.HandleFunc("POST /sessions/{session_id}/open", h.openSessionByID)
	// r.HandleFunc("POST /sessions/{session_id}/close", h.closeSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/start", h.startSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/end", h.endSessionByID)

	r.HandleFunc("POST /sessions/{session_id}/examinees/{examinee_id}/readmit", h.readmitExaminee)
	r.HandleFunc("POST /sessions/{session_id}/examinees/{examinee_id}/unflag", h.unflagExaminee)
	r.HandleFunc("POST /sessions/{session_id}/submit", h.submitBySessionID)

	r.HandleFunc("GET /sessions/{session_id}/examinees", h.getExamineesInSession)

	// POST /sessions/{session_id}/submit					(For examinees will submit the submission for that session)
	// PUT	/sessions/{session_id}/answers/{question_id}	(For examines will change the answer of a given question)
	// GET	/sessions/{session_id} 							(For examinees will produce their submission if they have one)

	r.HandleFunc("POST /sessions/{session_id}/mark", h.markSessionByID)

}

// -----------------------------------------------------------------------
// --- In-Session Related ------------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) enrolInSession(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	var body JoinSessionBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	submission, err := h.svc.EnrolWithCode(r.Context(), user, body.JoinCode)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submission": submission}, nil)
}

func (h *SessionHandler) submitBySessionID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	if err := h.svc.SubmitForSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) openSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
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
		return
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
		return
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
		return
	}

	if err := h.svc.EndSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) unflagExaminee(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	examineeID, err := uuid.Parse(r.PathValue("examinee_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	if err := h.svc.UnflagExaminee(r.Context(), user, sessionID, examineeID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) readmitExaminee(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	examineeID, err := uuid.Parse(r.PathValue("examinee_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	if err := h.svc.ReadmitExaminee(r.Context(), user, sessionID, examineeID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// -----------------------------------------------------------------------
// --- Post-Session Related ----------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) markSessionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	if err := h.svc.MarkSubmissionsForSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) getExamineesInSession(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID", nil)
		return
	}

	examinees, err := h.svc.FindSubmissionsForSession(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"examinees": examinees}, nil)
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

	session, err := h.svc.CreateSession(r.Context(), user, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"session": session}, nil)
}

func (h *SessionHandler) getSessions(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	query := r.URL.Query()

	search := query.Get("search")
	status := store.SessionStatus(query.Get("status"))
	cursor, err := api.DecodeCursor(query.Get("cursor"))
	if err != nil {
		cursor = nil
	}

	const size int32 = 10
	sessions, count, err := h.svc.FindSessions(r.Context(), user, cursor, size+1, search, status)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	if len(sessions) > int(size) {
		last := sessions[size-1]
		next := api.Cursor{Ts: last.CreatedAt, ID: last.ID}

		json.WriteJSON(w, http.StatusOK, json.Wrapper{
			"sessions": sessions[:size],
			"count":    count,
			"next":     api.EncodeCursor(next),
		}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{
		"sessions": sessions,
		"count":    count,
	}, nil)
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

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"session": session}, nil)
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
