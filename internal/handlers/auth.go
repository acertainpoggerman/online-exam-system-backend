package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/models"
)

// Register as a new User. Returns a token on Success
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {

	var reqBody models.RegisterRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	tokenString, err := h.service.Register(r.Context(), reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"token": tokenString}
	api.WriteJSON(w, http.StatusCreated, res, nil)
}

// Log-in to the API. Receives a token on Success
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {

	var reqBody models.LoginRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	tokenString, err := h.service.Login(r.Context(), reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"token": tokenString}
	api.WriteJSON(w, http.StatusOK, res, nil)
}
