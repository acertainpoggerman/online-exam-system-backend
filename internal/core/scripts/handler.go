package scripts

import (
	"log"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/google/uuid"
)

type ScriptHandler struct {
	svc ScriptService
}

func NewScriptHandler(svc ScriptService) *ScriptHandler {
	return &ScriptHandler{svc}
}

func (h *ScriptHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("POST /scripts", h.postScript)
	r.HandleFunc("GET /scripts", h.getScripts)
	r.HandleFunc("GET /scripts/{script_id}", h.getScriptByID)
	r.HandleFunc("PUT /scripts/{script_id}", h.putScript)
	// r.HandleFunc("PATCH /scripts/{script_id}", h.patchScriptByID)
	r.HandleFunc("DELETE /scripts/{script_id}", h.deleteScriptByID)

	r.HandleFunc("POST /scripts/{script_id}/questions", h.postQuestion)
	r.HandleFunc("PUT /scripts/{script_id}/questions/{question_id}", h.putQuestion)
	r.HandleFunc("DELETE /scripts/{script_id}/questions/{question_id}", h.deleteQuestion)
}

func (h *ScriptHandler) postScript(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	var body CreateScriptBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	script, err := h.svc.CreateScript(r.Context(), user, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		log.Println(err)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"script": script}, nil)
}

func (h *ScriptHandler) getScripts(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "User ID not found", nil)
		return
	}

	query := r.URL.Query()

	search := query.Get("search")
	cursor, err := api.DecodeCursor(query.Get("cursor"))
	if err != nil {
		cursor = nil
	}

	const size int32 = 10
	scripts, count, err := h.svc.FindScripts(r.Context(), user, cursor, size+1, search)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	if len(scripts) > int(size) {
		last := scripts[size-1]
		next := api.Cursor{Ts: last.LastModifiedAt, ID: last.ID}

		json.WriteJSON(w, http.StatusOK, json.Wrapper{
			"scripts": scripts[:size],
			"count":   count,
			"next":    api.EncodeCursor(next),
		}, nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{
		"scripts": scripts,
		"count":   count,
	}, nil)
}

func (h *ScriptHandler) getScriptByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "User ID not found"}, nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	script, err := h.svc.FindScriptByID(r.Context(), user, scriptID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"script": script}, nil)
}

func (h *ScriptHandler) putScript(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "User ID not found"}, nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	var body CreateScriptBody
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := h.svc.UpdateScriptFields(r.Context(), user, scriptID, body); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// func (h *ScriptHandler) patchScriptByID(w http.ResponseWriter, r *http.Request) {

// 	user, err := jwt.GetUserDataFromContext(r.Context())
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "User ID not found"}, nil)
// 		return
// 	}

// 	scriptID, err := uuid.Parse(r.PathValue("script_id"))
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err.Error()}, nil)
// 		return
// 	}

// 	var body UpdateScriptBody
// 	if err := json.ReadRequestBody(r, &body); err != nil {
// 		json.WriteJSON(w, http.StatusBadRequest, err.Error(), nil)
// 		return
// 	}

// 	script, err := h.svc.UpdateScriptByID(r.Context(), user, scriptID, body.IncludeScript, body.Actions)
// 	if err != nil {
// 		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
// 		return
// 	}

// 	if script != nil {
// 		json.WriteJSON(w, http.StatusOK, json.Wrapper{"script": script}, nil)
// 		return
// 	}

// 	json.WriteJSON(w, http.StatusNoContent, nil, nil)
// }

func (h *ScriptHandler) deleteScriptByID(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "User ID not found"}, nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	if err := h.svc.DeleteScriptByID(r.Context(), user, scriptID); err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}

// ------------------------------------------------------------------------------------
// --- Question Endpoints -------------------------------------------------------------
// ------------------------------------------------------------------------------------

func (h *ScriptHandler) postQuestion(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	var body Question
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err, nil)
		return
	}

	question, err := h.svc.CreateQuestion(r.Context(), user, scriptID, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusCreated, json.Wrapper{"question": question}, nil)
}

func (h *ScriptHandler) putQuestion(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	questionID, err := uuid.Parse(r.PathValue("question_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	var body Question
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, err, nil)
		return
	}

	log.Println(body.AnswerKey)

	question, err := h.svc.ReplaceQuestion(r.Context(), user, scriptID, questionID, body)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, json.Wrapper{"question": question}, nil)
}

func (h *ScriptHandler) deleteQuestion(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	scriptID, err := uuid.Parse(r.PathValue("script_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	questionID, err := uuid.Parse(r.PathValue("question_id"))
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": err}, nil)
		return
	}

	question, err := h.svc.DeleteQuestion(r.Context(), user, scriptID, questionID)
	if err != nil {
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": err.Error()}, nil)
		return
	}

	json.WriteJSON(w, http.StatusNoContent, json.Wrapper{"question": question}, nil)
}
