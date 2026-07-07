package submissions

import (
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
)

type SubmissionHandler struct {
	svc SubmissionService
}

func NewSubmissionHandler(svc SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc}
}

func (h *SubmissionHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /submissions/{submission_id}", h.getSubmissionByID)
	r.HandleFunc("PUT /submissions/{submission_id}/answers/{question_id}", h.putAnswerForQuestion)
}

func (h *SubmissionHandler) getSubmissionByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	submissionID, err := uuid.Parse(r.PathValue("submission_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	submission, err := h.svc.FindSubmissionByID(r.Context(), user, submissionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, submission, nil)
}

func (h *SubmissionHandler) putAnswerForQuestion(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	var body Answer
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	submissionID, err := uuid.Parse(r.PathValue("submission_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	questionID, err := uuid.Parse(r.PathValue("question_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	if err := h.svc.UpdateSubAnswerForQuestion(r.Context(), user, submissionID, questionID, body); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}
