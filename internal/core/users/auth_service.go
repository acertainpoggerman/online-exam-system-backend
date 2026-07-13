package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/acertainpoggerman/online-exam-system/internal/jwt"
	"github.com/acertainpoggerman/online-exam-system/internal/passwords"
	"github.com/jackc/pgx/v5/pgconn"
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

	hash, err := passwords.GeneratePasswordHash(password)
	if err != nil {
		return "", err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// Add Basic User Details

	user, err := qtx.CreateUser(ctx, store.CreateUserParams{
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", &apperr.FieldError{
				Field:   "email",
				Message: "Email already in use",
				Code:    "EMAIL_TAKEN",
			}
		}
	}

	// Add Specific "Role"

	switch role {
	case store.UserRoleExaminee:
		_, err = qtx.CreateExaminee(ctx, user.ID)
		if err != nil {
			return "", err
		}
	case store.UserRoleExaminer:
		_, err = qtx.CreateExaminer(ctx, user.ID)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("Requested user role: %v does not exist", role)
	}

	// Generate JWT for User

	tokenString, err := jwt.CreateJWT(
		user,
		svc.jwtSecretKey,
		svc.jwtExpiryTime,
	)
	if err != nil {
		return "", err
	}

	return tokenString, tx.Commit(ctx)
}

func (svc *authService) LoginUser(ctx context.Context, email string, password string) (string, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// Find the user by email (emails are set to be unique in the pool)

	user, err := qtx.FindUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	// Check given password is correct

	if !passwords.VerifyPassword(password, user.PasswordHash) {
		return "", fmt.Errorf("Wrong username / password")
	}

	// Check role in database

	switch user.Role {
	case store.UserRoleExaminee:
	case store.UserRoleExaminer:
	default:
	}

	// Generate JWT for User

	tokenString, err := jwt.CreateJWT(
		user,
		svc.jwtSecretKey,
		svc.jwtExpiryTime,
	)
	if err != nil {
		return "", err
	}

	return tokenString, tx.Commit(ctx)
}
