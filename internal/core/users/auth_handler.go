package users

import (
	"errors"
	"net/http"

	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/acertainpoggerman/online-exam-system/internal/json"
)

type AuthHandler struct {
	svc    AuthService
	jwtAge int
}

func NewAuthHandler(svc AuthService, jwtAge int) *AuthHandler {
	return &AuthHandler{svc, jwtAge}
}

func (h *AuthHandler) RegisterRoutes(r *http.ServeMux) {
	r.HandleFunc("POST /register", h.registerUser)
	r.HandleFunc("POST /login", h.loginUser)
	r.HandleFunc("POST /logout", h.logoutUser)
}

func (h *AuthHandler) registerUser(w http.ResponseWriter, r *http.Request) {

	var body RegisterUserRequest
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "Failed to process request body"}, nil)
		return
	}

	token, err := h.svc.RegisterUser(
		r.Context(),
		body.FirstName,
		body.LastName,
		body.Email,
		body.Password,
		body.UserRole,
	)
	if err != nil {
		var fieldErr *apperr.FieldError
		if errors.As(err, &fieldErr) {
			json.WriteJSON(w, http.StatusUnprocessableEntity, json.Wrapper{"error": fieldErr}, nil)
			return
		}
		json.WriteJSON(w, http.StatusInternalServerError, json.Wrapper{"error": "Failed to register user"}, nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.jwtAge,
		Path:     "/",
	})

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"token": token}, nil)
}

func (h *AuthHandler) loginUser(w http.ResponseWriter, r *http.Request) {

	var body LoginUserRequest
	if err := json.ReadRequestBody(r, &body); err != nil {
		json.WriteJSON(w, http.StatusBadRequest, json.Wrapper{"error": "Failed to process request body"}, nil)
		return
	}

	token, err := h.svc.LoginUser(r.Context(), body.Email, body.Password)
	if err != nil {
		json.WriteJSON(w, http.StatusUnauthorized, json.Wrapper{"error": "Invalid username or password"}, nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.jwtAge,
		Path:     "/",
	})

	json.WriteJSON(w, http.StatusOK, json.Wrapper{"token": token}, nil)
}

// Cookie-only. For clearing the cookies on the
// browser side. Always sends http.StatusNoContent (204)
func (h *AuthHandler) logoutUser(w http.ResponseWriter, r *http.Request) {

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Path:     "/",
	})

	json.WriteJSON(w, http.StatusNoContent, nil, nil)
}
