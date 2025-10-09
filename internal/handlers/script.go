package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
)

func (h *Handler) HandleCreateScript(w http.ResponseWriter, r *http.Request) {

	var reqBody models.CreateScriptRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	creator, ok := r.Context().Value(api.AuthUserData).(*tools.UserData)
	if !ok {
		api.BadRequestResponse(w, r, fmt.Errorf("failed to convert user data"))
		return
	}

	script, err := h.service.CreateScript(r.Context(), creator.ID, reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"script": script}
	api.WriteJSON(w, http.StatusCreated, res, nil)
}

func (h *Handler) HandleGetScriptByID(w http.ResponseWriter, r *http.Request) {

	scriptID := r.PathValue("id")
	script, err := h.service.GetScriptByID(r.Context(), scriptID)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"script": script}
	api.WriteJSON(w, http.StatusOK, res, nil)
}

func (h *Handler) HandleDeleteScriptByID(w http.ResponseWriter, r *http.Request) {

	scriptID := r.PathValue("id")
	if err := h.service.DeleteScriptByID(r.Context(), scriptID); err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusNoContent, "", nil)
}

func (h *Handler) HandleUpdateScriptByID(w http.ResponseWriter, r *http.Request) {

	scriptID := r.PathValue("id")

	var reqBody models.UpdateScriptRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	script, err := h.service.UpdateScriptByID(r.Context(), scriptID, reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"script": script}
	api.WriteJSON(w, http.StatusCreated, res, nil)
}
