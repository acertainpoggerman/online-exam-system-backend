package handlers

import (
	"fmt"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
)

// Get Information of the Authenticated User.
func (h *Handler) HandleGetMe(w http.ResponseWriter, r *http.Request) {

	userData, ok := r.Context().Value(api.AuthUserData).(*tools.UserData)
	if !ok {
		api.BadRequestResponse(w, r, fmt.Errorf("failed to convert user data"))
		return
	}

	user, err := h.service.GetUserByID(r.Context(), userData.ID)
	if err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	res := api.Envelope{"user": user}
	api.WriteJSON(w, http.StatusOK, res, nil)
}

// "Delete" User from the System (Recoverable)
func (h *Handler) HandleDeleteMe(w http.ResponseWriter, r *http.Request) {

	userData, ok := r.Context().Value(api.AuthUserData).(*tools.UserData)
	if !ok {
		api.BadRequestResponse(w, r, fmt.Errorf("failed to convert user data"))
		return
	}

	if err := h.service.DeleteUserByID(r.Context(), userData.ID); err != nil {
		api.BadRequestResponse(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusNoContent, "", nil)
}
