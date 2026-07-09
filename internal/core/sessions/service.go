package sessions

import (
	"context"
	"errors"
	"fmt"
	"log"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/acertainpoggerman/online-exam-system/internal/core/submissions"
	"github.com/acertainpoggerman/online-exam-system/internal/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionService interface {
	CreateSession(ctx context.Context, user store.User, data CreateSessionBody) (string, error)
	FindSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID) (Session, error)
	FindSessions(ctx context.Context, user store.User) ([]Session, error)
	UpdateSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID, data CreateSessionBody) (Session, error)

	JoinSessionByCode(ctx context.Context, user store.User, joinCode string) (submissions.Submission, error)

	OpenSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	CloseSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	StartSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	EndSession(ctx context.Context, user store.User, sessionID uuid.UUID) error

	ExamineeInSession(ctx context.Context, user store.User, sessionID uuid.UUID) (bool, error)
	OnGraceExpired(userID, sessionID uuid.UUID)
	SendStateSync(ctx context.Context, userID, sessionID uuid.UUID) error

	MarkSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
}

type ExtSessionService interface {
	FindSessionByID(ctx context.Context, user store.User, sessionID uuid.UUID) (Session, error)
}

type sessionService struct {
	q    *store.Queries
	pool *pgxpool.Pool
	hub  *Hub
	sub  submissions.ExtSubmissionService
}

func NewSessionService(
	q *store.Queries,
	pool *pgxpool.Pool,
	hub *Hub,
	sub submissions.ExtSubmissionService,
) *sessionService {
	return &sessionService{q, pool, hub, sub}
}

// ------------------------------------------------------------------------------------
// --- Session Participation Services -------------------------------------------------
// ------------------------------------------------------------------------------------

func (svc *sessionService) JoinSessionByCode(ctx context.Context, user store.User, joinCode string) (submissions.Submission, error) {

	if user.Role != store.UserRoleExaminee {
		return submissions.Submission{}, fmt.Errorf("Role not allowed: %s", user.Role)
	}

	session, err := svc.q.FindSessionByJoinCode(ctx, joinCode)
	if err != nil {
		return submissions.Submission{}, err
	}

	return svc.sub.CreateSubmission(ctx, user, submissions.CreateSubmissionBody{
		SessionID: session.ID,
	})
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

	// ---------------------------------------------------------------------
	// --- Updating session status -----------------------------------------

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	started, err := qtx.StartSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if err := qtx.SetQuestionsForSubmissions(ctx, sessionID); err != nil {
		return err
	}

	if err := qtx.SetSubmissionsEditableForSession(ctx, sessionID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// ---------------------------------------------------------------------
	// --- Broadcasting Start Event ----------------------------------------

	svc.hub.Broadcast(sessionID, Message{
		Type: MessageTypeSessionStarted,
		Data: json.Wrapper{
			"session_id": sessionID,
			"started_at": started.StartedAt,
		},
	})

	return nil
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

	// ------------------------------------------------------------------------------------
	// --- Updating session status --------------------------------------------------------

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	if _, err := qtx.EndSession(ctx, sessionID); err != nil {
		return err
	}

	if err := qtx.SubmitAllSubmissionsForSession(ctx, sessionID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// ------------------------------------------------------------------------------------
	// --- Broadcasting End Event ---------------------------------------------------------

	svc.hub.Broadcast(sessionID, Message{
		Type: MessageTypeSessionEnded,
	})
	svc.hub.CloseSession(sessionID)

	return nil
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

	// ------------------------------------------------------------------------------------
	// --- Updating session status --------------------------------------------------------

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

// ------------------------------------------
// --- WS Related ---------------------------
// ------------------------------------------

func (svc *sessionService) ExamineeInSession(ctx context.Context, user store.User, sessionID uuid.UUID) (bool, error) {

	if user.Role != store.UserRoleExaminee {
		return false, nil
	}

	if _, err := svc.q.FindActiveSubmissionInSession(ctx, store.FindActiveSubmissionInSessionParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // Row not found
			return false, nil
		}
		return false, err // DB Failure
	}

	return true, nil
}

func (svc *sessionService) OnGraceExpired(userID, sessionID uuid.UUID) {

	ctx := context.Background()

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return
	}

	if session.Status != store.SessionStatusStarted {
		log.Printf(
			"Session:(%s) failed to reconnect. Session has not been started, no issue",
			sessionID,
		)
		return
	}

	log.Printf(
		"Session:(%s) failed to reconnect. Does not submit script for now",
		sessionID,
	)
}

func (svc *sessionService) SendStateSync(ctx context.Context, userID, sessionID uuid.UUID) error {

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	svc.hub.SendToUser(userID, sessionID, Message{
		Type: MessageTypeSessionStateSync,
		Data: json.Wrapper{
			"status": session.Status,
		},
	})

	return nil
}

// ------------------------------------------------------------------------------------
// --- Marking Services ---------------------------------------------------------------
// ------------------------------------------------------------------------------------

func (svc *sessionService) MarkSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuring User (as Examiner) owns the Script ----------------------------------------

	if session, err := qtx.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if user.ID != session.CreatorID {
		return fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Marking Submissions -----------------------------------------------------------------

	submissions, err := qtx.FindSubmissionsForSession(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, submission := range submissions {
		if err := svc.sub.MarkSubmission(ctx, qtx, submission.ID); err != nil {
			return err
		}
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

	// -------------------------------------------------------------
	// --- Ensuring examiner owns the session ----------------------

	if session, err := qtx.FindSessionByID(ctx, sessionID); err != nil {
		return Session{}, err
	} else if session.CreatorID != user.ID {
		return Session{}, fmt.Errorf("Cannot modify resource")
	}

	// -------------------------------------------------------------
	// --- Modifying session details -------------------------------

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
