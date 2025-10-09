package repository

import (
	"context"
	"fmt"

	"github.com/acertainpoggerman/online-exam-system/internal/models"
)

const (
	CollectionNameUsers    = "users"
	CollectionNameScripts  = "scripts"
	CollectionNameSessions = "sessions"
)

func NotImplementedError(method string) error {
	return fmt.Errorf("Repository Method not implemented: %v", method)
}

type Repository interface {

	// [User Methods]
	CreateUser(ctx context.Context, user models.User) (string, error)
	FindUsers(ctx context.Context) ([]models.User, error)
	FindUserByID(ctx context.Context, userID string) (*models.User, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	DeleteUserByID(ctx context.Context, userID string) error

	// [Script Methods]
	CreateScript(ctx context.Context, script models.Script) (*models.Script, error)
	FindScripts(ctx context.Context) ([]models.Script, error)
	FindScriptByID(ctx context.Context, scriptID string) (*models.Script, error)
	DeleteScriptByID(ctx context.Context, scriptID string) error
	UpdateScript(ctx context.Context, script models.Script) (*models.Script, error)

	// [Sesssion Methods]
	CreateSession(ctx context.Context, session models.Session) (*models.Session, error)
	FindSessions(ctx context.Context) ([]models.Session, error)
	FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error)
	DeleteSessionByID(ctx context.Context, sessionID string) error
	UpdateSession(ctx context.Context, session models.Session) (*models.Session, error)
}
