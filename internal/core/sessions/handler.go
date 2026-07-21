package sessions

import (
	"errors"
	"log"
	"net/http"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
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

	// r.HandleFunc("POST /sessions/{session_id}/close", h.closeSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/open", h.openSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/start", h.startSessionByID)
	r.HandleFunc("POST /sessions/{session_id}/end", h.endSessionByID)

	r.HandleFunc("POST /sessions/{session_id}/examinees/{examinee_id}/readmit", h.readmitExaminee)
	r.HandleFunc("POST /sessions/{session_id}/examinees/{examinee_id}/unflag", h.unflagExaminee)
	r.HandleFunc("GET /sessions/{session_id}/examinees/{examinee_id}/logs", h.getExamineeLogs)
	r.HandleFunc("GET /sessions/{session_id}/examinees", h.getSubmissionsInSession)
	// r.HandleFunc("GET /sessions/{session_id}/examinees/{examinee_id}", h.getSubmission)

	r.HandleFunc("PUT /sessions/{session_id}/answers/{question_id}", h.putAnswer)
	r.HandleFunc("POST /sessions/{session_id}/submit", h.submitForSession)

	r.HandleFunc("POST /sessions/{session_id}/mark", h.autoMarkSession)
	r.HandleFunc("POST /sessions/{session_id}/examinees/{examinee_id}/mark", h.examinerMarkSubmission)
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

func (h *SessionHandler) putAnswer(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	questionID, err := uuid.Parse(r.PathValue("question_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	var answer Answer
	if err := json.ReadRequestBody(r, &answer); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	if err := h.svc.UpdateAnswer(r.Context(), user, sessionID, questionID, answer); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) submitForSession(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("session_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
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

func (h *SessionHandler) getExamineeLogs(w http.ResponseWriter, r *http.Request) {

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

	logs, err := h.svc.FindExamineeLogs(r.Context(), user, sessionID, examineeID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"logs": logs}, nil)
}

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

func (h *SessionHandler) autoMarkSession(w http.ResponseWriter, r *http.Request) {

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

	if err := h.svc.AutoMarkSession(r.Context(), user, sessionID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

func (h *SessionHandler) getSubmissionsInSession(w http.ResponseWriter, r *http.Request) {

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

func (h *SessionHandler) examinerMarkSubmission(w http.ResponseWriter, r *http.Request) {

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

	var body MarkForExamineeBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": apperr.ErrBadRequest}, nil)
		return
	}

	if err := h.svc.ExaminerMarkSubmission(r.Context(), user, sessionID, examineeID, body); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
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

	switch user.Role {
	case store.UserRoleExaminer:
		h.examinerGetSessions(w, r, user)
	case store.UserRoleExaminee:
		h.examineeGetSubmissions(w, r, user)
	}
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) examinerGetSessions(w http.ResponseWriter, r *http.Request, user store.User) {

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

func (h *SessionHandler) examineeGetSubmissions(w http.ResponseWriter, r *http.Request, user store.User) {

	submissions, err := h.svc.ExamineeFindSubmissions(r.Context(), user)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submissions": submissions}, nil)
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

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

	switch user.Role {
	case store.UserRoleExaminer:
		h.examinerGetSession(w, r, user, sessionID)
	case store.UserRoleExaminee:
		h.examineeGetSubmission(w, r, user, sessionID)
	}
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

func (h *SessionHandler) examinerGetSession(w http.ResponseWriter, r *http.Request, user store.User, sessionID uuid.UUID) {

	log.Println("Examiner here")

	session, err := h.svc.FindSessionByID(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"session": session}, nil)
}

func (h *SessionHandler) examineeGetSubmission(w http.ResponseWriter, r *http.Request, user store.User, sessionID uuid.UUID) {

	log.Println("Examinee here")

	submission, err := h.svc.ExamineeFindSubmission(r.Context(), user, sessionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submission": submission}, nil)
}

// -----------------------------------------------------------------------
// -----------------------------------------------------------------------

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

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"session": session}, nil)
}
