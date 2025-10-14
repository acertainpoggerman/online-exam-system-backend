package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
)

func (h *Handler) HandleCreateSession(w http.ResponseWriter, r *http.Request) {

	var reqBody models.CreateSessionRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		fmt.Println(err)
		api.BadRequestResponse(w, r, err)
		return
	}

	creator, ok := r.Context().Value(api.AuthUserData).(*tools.UserData)
	if !ok {
		fmt.Println("In getting user data")
		api.BadRequestResponse(w, r, fmt.Errorf("failed to convert user data"))
		return
	}

	session, err := h.service.CreateSession(r.Context(), creator.ID, reqBody)
	if err != nil {
		fmt.Println("In creating session")
		return
	}

	res := api.Envelope{"session": session}
	api.WriteJSON(w, http.StatusCreated, res, nil)
}

func (h *Handler) HandleGetSessionByID(w http.ResponseWriter, r *http.Request) {

	sessionID := r.PathValue("id")
	session, err := h.service.GetSessionByID(r.Context(), sessionID)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"session": session}
	api.WriteJSON(w, http.StatusOK, res, nil)
}

func (h *Handler) HandleDeleteSessionByID(w http.ResponseWriter, r *http.Request) {

	sessionID := r.PathValue("id")
	if err := h.service.DeleteSessionByID(r.Context(), sessionID); err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusNoContent, "", nil)
}

func (h *Handler) HandleUpdateSessionByID(w http.ResponseWriter, r *http.Request) {

	sessionID := r.PathValue("id")

	var reqBody models.UpdateSessionRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	session, err := h.service.UpdateSessionByID(r.Context(), sessionID, reqBody)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"session": session}
	api.WriteJSON(w, http.StatusCreated, res, nil)
}

func (h *Handler) HandleConnectToSession(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) HandleStartSessionByID(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) HandleEndSessionByID(w http.ResponseWriter, r *http.Request) {

}
