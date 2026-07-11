package users

import (
	"context"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService interface {
	Me(ctx context.Context, user store.User) (store.User, error)
}

type userService struct {
	q    *store.Queries
	pool *pgxpool.Pool
}

func NewUserService(q *store.Queries, pool *pgxpool.Pool) *userService {
	return &userService{q, pool}
}

func (svc *userService) Me(ctx context.Context, user store.User) (store.User, error) {
	user, err := svc.q.FindUserByID(ctx, user.ID)
	if err != nil {
		return store.User{}, apperr.ErrNotFound
	}
	return user, nil
}
