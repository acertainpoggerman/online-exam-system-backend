package users

import store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"

// RegisterUser DTO Model

type RegisterUserRequest struct {
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Email     string         `json:"email"`
	Password  string         `json:"password"`
	UserRole  store.UserRole `json:"role"`
}

// LoginUser DTO Model

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
