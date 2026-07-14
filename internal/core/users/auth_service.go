package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/acertainpoggerman/online-exam-system/internal/passwords"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthService interface {
	RegisterUser(ctx context.Context, firstName string, lastName string, email string, password string, role store.UserRole) (string, error)
	LoginUser(ctx context.Context, email string, password string) (string, error)
}

type authService struct {
	q             *store.Queries
	pool          *pgxpool.Pool
	jwtSecretKey  []byte
	jwtExpiryTime time.Duration
}

func NewAuthService(q *store.Queries, pool *pgxpool.Pool, jwtSecretKey []byte, jwtExpiryTime time.Duration) *authService {
	return &authService{q, pool, jwtSecretKey, jwtExpiryTime}
}

const minPasswordLength = 8

// -----------------------------------------------------------------------
// --- Implementing authService Interface --------------------------------
// -----------------------------------------------------------------------

func (svc *authService) RegisterUser(
	ctx context.Context,
	firstName string,
	lastName string,
	email string,
	password string,
	role store.UserRole,
) (string, error) {

	// Hash Password

	if len(password) < minPasswordLength {
		message := fmt.Sprintf("Password must be at least %d characters", minPasswordLength)
		return "", &apperr.FieldError{
			Fields:  []string{"password"},
			Message: message,
			Code:    "minLength",
		}
	}

	hash, err := passwords.GeneratePasswordHash(password)
	if err != nil {
		log.Printf("Error in RegisterUser: (%v)\n", err)
		return "", apperr.ErrInternal
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		log.Printf("Error in RegisterUser: (%v)\n", err)
		return "", apperr.ErrInternal
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// -----------------------------------------------------------------
	// --- Creating Based on Role --------------------------------------
	// -----------------------------------------------------------------

	var user store.User
	switch role {
	// -------------------------------------------------
	case store.UserRoleExaminee:
		result, err := qtx.CreateExaminee(ctx, store.CreateExamineeParams{
			FirstName:    firstName,
			LastName:     lastName,
			Email:        email,
			PasswordHash: hash,
		})
		user = store.User(result)
		if err != nil {
			return "", svc.resolveRegisterError(ctx, err)
		}
	// -------------------------------------------------
	case store.UserRoleExaminer:
		result, err := qtx.CreateExaminer(ctx, store.CreateExaminerParams{
			FirstName:    firstName,
			LastName:     lastName,
			Email:        email,
			PasswordHash: hash,
		})
		user = store.User(result)
		if err != nil {
			return "", svc.resolveRegisterError(ctx, err)
		}
	// -------------------------------------------------
	default:
		log.Println("Error in RegisterUser: (could not determine role of user)")
		return "", apperr.ErrBadRequest
	}

	// -----------------------------------------------------------------
	// --- Generate JWT for User ---------------------------------------
	// -----------------------------------------------------------------

	tokenString, err := jwt.CreateJWT(
		user,
		svc.jwtSecretKey,
		svc.jwtExpiryTime,
	)
	if err != nil {
		log.Printf("Error in RegisterUser: (%v)\n", err)
		return "", apperr.ErrInternal
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error in RegisterUser: could not commit (%v)\n", err)
		return "", apperr.ErrInternal
	}

	return tokenString, nil
}

func (svc *authService) LoginUser(ctx context.Context, email string, password string) (string, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// -----------------------------------------------------------------
	// --- Find User Email ---------------------------------------------
	// -----------------------------------------------------------------

	user, err := qtx.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &apperr.FieldError{
				Fields:  []string{"email", "password"},
				Message: "Invalid email or password",
				Code:    "INVALID_CREDENTIALS",
			}
		}
		log.Printf("Error in LoginUser: could not find email (%v)\n", err)
		return "", apperr.ErrInternal
	}

	// -----------------------------------------------------------------
	// --- Check Password ----------------------------------------------
	// -----------------------------------------------------------------

	if !passwords.VerifyPassword(password, user.PasswordHash) {
		return "", &apperr.FieldError{
			Fields:  []string{"email", "password"},
			Message: "Invalid email or password",
			Code:    "INVALID CREDENTIALS",
		}
	}

	// -----------------------------------------------------------------
	// --- Generate JWT for User ---------------------------------------
	// -----------------------------------------------------------------

	tokenString, err := jwt.CreateJWT(
		user,
		svc.jwtSecretKey,
		svc.jwtExpiryTime,
	)
	if err != nil {
		return "", apperr.ErrInternal
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error in RegisterUser: could not commit (%v)", err)
		return "", apperr.ErrInternal
	}

	return tokenString, nil
}
