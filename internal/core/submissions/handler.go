package submissions

import (
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
)

type SubmissionHandler struct {
	svc SubmissionService
}

func NewSubmissionHandler(svc SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc}
}

func (h *SubmissionHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /submissions", h.getSubmissions)
	// r.HandleFunc("GET /sessions/{session_id}/submissions", h.getSubmissionsForSession)

	// r.HandleFunc("GET /submissions/{submission_id}", h.getSubmissionByID)
	r.HandleFunc("POST /submissions", h.postSubmission)
	// r.HandleFunc("PATCH /submissions/{submission_id}", h.patchSubmission)
}

func (h *SubmissionHandler) getSubmissions(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	submissions, err := h.svc.FindSubmissions(r.Context(), user)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submissions": submissions}, nil)
}

// func (h *SubmissionHandler) getSubmissionsForSession(w http.ResponseWriter, r *http.Request) {

// 	user, err := jwt.GetUserDataFromContext(r.Context())
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
// 		return
// 	}

// 	sessionID, err := uuid.Parse(r.PathValue("session_id"))
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Invalid session ID passed", nil)
// 		return
// 	}

// 	submissions, err := h.svc.FindSubmissionsForSession(r.Context(), user, sessionID)
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
// 		return
// 	}

// 	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submissions": submissions}, nil)
// }

// func (h *SubmissionHandler) getSubmissionByID(w http.ResponseWriter, r *http.Request) {

// 	user, err := jwt.GetUserDataFromContext(r.Context())
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
// 		return
// 	}

// 	submissionID, err := uuid.Parse(r.PathValue("submission_id"))
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, "Invalid submission ID passed", nil)
// 		return
// 	}

// 	submission, err := h.svc.FindSubmissionByID(r.Context(), user, submissionID)
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
// 		return
// 	}

// 	json.WriteJSON(w, http.StatusOK, json.Wrapper{"submission": submission}, nil)
// }

func (h *SubmissionHandler) postSubmission(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	var body CreateSubmissionBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	id, err := h.svc.CreateSubmission(r.Context(), user, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"submission_id": id}, nil)
}
