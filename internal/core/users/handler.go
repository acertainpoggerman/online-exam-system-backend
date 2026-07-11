package users

import (
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
)

type UserHandler struct {
	svc UserService
}

func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{svc}
}

func (h *UserHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /me", h.getMe)
}

func (h *UserHandler) getMe(w http.ResponseWriter, r *http.Request) {

	user, err := jwt.GetUserDataFromContext(r.Context())
	if err != nil {
		json.WriteJSON(w, http.StatusBadRequest, "Could not get user", nil)
		return
	}

	fullUser, err := h.svc.Me(r.Context(), user)
	json.WriteJSON(w, http.StatusOK, json.Wrapper{"user": fullUser}, nil)
}
