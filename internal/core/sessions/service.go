package sessions

import (
	"context"
	"fmt"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type SessionService interface {
	CreateSession(ctx context.Context, user store.User, data CreateSessionBody) (string, error)
	FindSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID) (Session, error)
	FindSessions(ctx context.Context, user store.User) ([]Session, error)
	UpdateSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID, data CreateSessionBody) (Session, error)

	JoinSessionByCode(ctx context.Context, user store.User, joinCode string) error

	OpenSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	CloseSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	StartSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	EndSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
}

type ExtSessionService interface {
	FindSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID) (Session, error)
}

type sessionService struct {
	q    *store.Queries
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewSessionService(q *store.Queries, pool *pgxpool.Pool, rdb *redis.Client) *sessionService {
	return &sessionService{q, pool, rdb}
}

// ------------------------------------------------------------------------------------
// --- Session Participation Services -------------------------------------------------
// ------------------------------------------------------------------------------------

func (svc *sessionService) JoinSessionByCode(ctx context.Context, user store.User, joinCode string) error {

	if user.Role != store.UserRoleExaminee {
		return fmt.Errorf("Role not allowed: %s", user.Role)
	}

	if _, err := svc.q.FindSessionByJoinCode(ctx, joinCode); err != nil {
		return err
	}

	return nil
}

func (svc *sessionService) StartSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Role not allowed: %s", user.Role)
	}

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.CreatorID != user.ID {
		return fmt.Errorf("User does not own resource")
	}

	// Updating session status

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	if _, err := qtx.StartSession(ctx, sessionID); err != nil {
		return err
	}

	// ------------------------------------------------------------------------------------
	// --- Publishing Session Event -------------------------------------------------------
	// ------------------------------------------------------------------------------------

	// event := SessionEvent{
	// 	Type: EventTypeSessionStarted,
	// 	Data: json.Wrapper{"script_id": session.ScriptID},
	// }
	// payload, err := json.Marshal(event)
	// if err != nil {
	// 	return err
	// }
	// if err := svc.rdb.Publish(ctx, sessionID.String(), payload).Err(); err != nil {
	// 	return err
	// }

	return tx.Commit(ctx)
}

func (svc *sessionService) EndSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Role not allowed: %s", user.Role)
	}

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.CreatorID != user.ID {
		return fmt.Errorf("User does not own resource")
	}

	// Updating session status

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	if _, err := qtx.EndSession(ctx, sessionID); err != nil {
		return err
	}

	// ------------------------------------------------------------------------------------
	// --- Publishing Session Event -------------------------------------------------------
	// ------------------------------------------------------------------------------------

	// event := SessionEvent{Type: EventTypeSessionEnded}
	// payload, err := json.Marshal(event)
	// if err != nil {
	// 	return err
	// }
	// if err := svc.rdb.Publish(ctx, sessionID.String(), payload).Err(); err != nil {
	// 	return err
	// }

	return tx.Commit(ctx)
}

func (svc *sessionService) OpenSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Role not allowed: %s", user.Role)
	}

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.CreatorID != user.ID {
		return fmt.Errorf("User does not own resource")
	}

	// Updating session status

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	if _, err := qtx.OpenSession(ctx, sessionID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (svc *sessionService) CloseSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Role not allowed: %s", user.Role)
	}

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.CreatorID != user.ID {
		return fmt.Errorf("User does not own resource")
	}

	// Updating session status

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	if _, err := qtx.CloseSession(ctx, sessionID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ------------------------------------------------------------------------------------
// --- CRUD Session Services ----------------------------------------------------------
// ------------------------------------------------------------------------------------

func (svc *sessionService) CreateSession(ctx context.Context, user store.User, data CreateSessionBody) (string, error) {

	if user.Role != store.UserRoleExaminer {
		return "", fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	id, err := qtx.CreateSession(ctx, store.CreateSessionParams{
		JoinCode:  generateSessionCode(10),
		CreatorID: user.ID,
		Title:     data.Title,
		ScriptID:  data.ScriptID,
	})
	if err != nil {
		return "", err
	}

	return id.String(), tx.Commit(ctx)
}

func (svc *sessionService) FindSessions(ctx context.Context, user store.User) ([]Session, error) {

	if user.Role != store.UserRoleExaminer {
		return []Session{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return []Session{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	sessions, err := qtx.FindSessionsForExaminer(ctx, user.ID)
	if err != nil {
		return []Session{}, err
	}

	return common.Map(sessions, func(s store.Session) Session {
		return Session{s}
	}), nil
}

func (svc *sessionService) FindSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID) (Session, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	session, err := qtx.FindSessionByID(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}

	return Session{session}, nil
}

func (svc *sessionService) UpdateSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID, data CreateSessionBody) (Session, error) {

	if user.Role != store.UserRoleExaminer {
		return Session{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// Ensuring examiner owns the session

	if session, err := qtx.FindSessionByID(ctx, sessionID); err != nil {
		return Session{}, err
	} else if session.CreatorID != user.ID {
		return Session{}, fmt.Errorf("Cannot modify resource")
	}

	// Modifying session details

	session, err := qtx.UpdateSessionFields(ctx, store.UpdateSessionFieldsParams{
		ID:       sessionID,
		Title:    data.Title,
		ScriptID: data.ScriptID,
	})
	if err != nil {
		return Session{}, err
	}

	return Session{session}, tx.Commit(ctx)
}
