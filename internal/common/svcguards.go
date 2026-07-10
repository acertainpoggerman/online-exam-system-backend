package common

import (
	"slices"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/google/uuid"
)

// Returns ErrForbidden if user's role does not match any of [roles]
func RequireRole(user store.User, roles ...store.UserRole) error {
	if !slices.Contains(roles, user.Role) {
		return apperr.ErrForbidden
	}
	return nil
}

// Returns ErrForbidden if user's ID does not match [ownerID]
func RequireOwner(user store.User, ownerID uuid.UUID) error {
	if ownerID != user.ID {
		return apperr.ErrForbidden
	}
	return nil
}
