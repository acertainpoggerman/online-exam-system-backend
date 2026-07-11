package sessions

import (
	"context"
	"errors"
	"log"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
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

	EnrolWithCode(ctx context.Context, user store.User, joinCode string) (submissions.Submission, error)
	SubmitForSession(ctx context.Context, user store.User, sessionID uuid.UUID) error

	OpenSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	CloseSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	StartSession(ctx context.Context, user store.User, sessionID uuid.UUID) error
	EndSession(ctx context.Context, user store.User, sessionID uuid.UUID) error

	ReadmitExaminee(ctx context.Context, user store.User, sessionID uuid.UUID, examineeID uuid.UUID) error
	UnflagExaminee(ctx context.Context, user store.User, sessionID uuid.UUID, examineeID uuid.UUID) error

	ExamineeCanConnect(ctx context.Context, user store.User, sessionID uuid.UUID) (bool, error)
	OnExamineeConnect(ctx context.Context, user store.User, sessionID uuid.UUID) error
	OnExamineeDisconnect(ctx context.Context, user store.User, sessionID uuid.UUID) error
	FindSubmissionStatus(ctx context.Context, user store.User, sessionID uuid.UUID) (store.SubmissionStatus, error)

	OnGraceExpired(ctx context.Context, userID, sessionID uuid.UUID) error
	SendStateSync(ctx context.Context, userID, sessionID uuid.UUID) error
	HandleProctorEvent(ctx context.Context, user store.User, event ProctorEvent) error

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

func (svc *sessionService) EnrolWithCode(ctx context.Context, user store.User, joinCode string) (submissions.Submission, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return submissions.Submission{}, err
	}

	session, err := svc.q.FindSessionByJoinCode(ctx, joinCode)
	if err != nil {
		return submissions.Submission{}, err
	}

	return svc.sub.CreateSubmission(ctx, user, submissions.CreateSubmissionBody{
		SessionID: session.ID,
	})
}

func (svc *sessionService) SubmitForSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	if _, err := svc.q.SetSubmissionSubmitted(ctx, store.SetSubmissionSubmittedParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrNotFound
		}
		log.Printf("Error SubmitSubmission: %v", err)
		return apperr.ErrInternal
	}

	svc.hub.RemoveMember(user.ID, sessionID)

	return nil
}

// ------------------------------------------
// ------------------------------------------

func (svc *sessionService) StartSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
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

	if _, err := qtx.SetSubmissionStatusesForStartedSession(ctx, sessionID); err != nil {
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

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
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

	if _, err := qtx.SubmitAllSubmissionsForSession(ctx, sessionID); err != nil {
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
	svc.hub.PurgeSession(sessionID)

	return nil
}

func (svc *sessionService) OpenSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
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

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
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
// --- Sub Status (Examiner Deps) -----------
// ------------------------------------------

func (svc *sessionService) ReadmitExaminee(ctx context.Context, user store.User, sessionID uuid.UUID, examineeID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrNotFound
		}
		log.Printf("Error in ReadmitExaminee: %v", err)
		return apperr.ErrInternal

	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err

	} else if err := common.RequireSessionHasStatus(
		session.Status,
		store.SessionStatusStarted,
	); err != nil {
		return err
	}

	if _, err := svc.q.SetSubmissionDisconnectedFromLeft(
		ctx, store.SetSubmissionDisconnectedFromLeftParams{
			SessionID:  sessionID,
			ExamineeID: examineeID,
		},
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrNotFound
		}
		log.Printf("Error in ReadmitExaminee: %v", err)
		return apperr.ErrInternal
	}

	return nil
}

func (svc *sessionService) UnflagExaminee(ctx context.Context, user store.User, sessionID uuid.UUID, examineeID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	if session, err := svc.q.FindSessionByID(ctx, sessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Error in UnflagExaminee: could not find session (%v)\n", err)
			return apperr.ErrNotFound
		}
		log.Printf("Error in UnflagExaminee: %v\n", err)
		return apperr.ErrInternal
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
	}

	// Needs to know the client state in the session's hub to make
	// decisions.
	//
	// - ConnStateConnected		: (Flagged -> Editable)
	// - ConnStateDisconnected 	: (Flagged -> Disconnected)
	// - ConnStateNone 			: (Flagged -> Disconnected)

	examineeState := svc.hub.GetMemberConnState(examineeID, sessionID)

	switch examineeState {
	// -----------------------------------------------------------
	case ConnStateConnected:
		if _, err := svc.q.SetSubmissionEditableFromFlagged(
			ctx, store.SetSubmissionEditableFromFlaggedParams{
				SessionID:  sessionID,
				ExamineeID: examineeID,
			},
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error in UnflagExaminee: could not set as editable (%v)\n", err)
				return apperr.ErrNotFound
			}
			log.Printf("Error in UnflagExaminee: %v\n", err)
			return apperr.ErrInternal
		}
	// -----------------------------------------------------------
	default:
		if _, err := svc.q.SetSubmissionDisconnectedFromFlagged(
			ctx, store.SetSubmissionDisconnectedFromFlaggedParams{
				SessionID:  sessionID,
				ExamineeID: examineeID,
			},
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Error in UnflagExaminee: could not set as editable (%v)\n", err)
				return apperr.ErrNotFound
			}
			log.Printf("Error in UnflagExaminee: %v\n", err)
			return apperr.ErrInternal
		}
	}

	svc.hub.SendToUser(sessionID, examineeID, Message{
		Type: MessageTypeSessionUnflagged,
	})

	return nil
}

// ------------------------------------------
// --- WS Related ---------------------------
// ------------------------------------------

func (svc *sessionService) FindSubmissionStatus(ctx context.Context, user store.User, sessionID uuid.UUID) (store.SubmissionStatus, error) {
	return svc.q.FindSubmissionStatus(ctx, store.FindSubmissionStatusParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
}

func (svc *sessionService) ExamineeCanConnect(ctx context.Context, user store.User, sessionID uuid.UUID) (bool, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return false, nil
	}

	if _, err := svc.q.FindConnectableSubmission(ctx, store.FindConnectableSubmissionParams{
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

func (svc *sessionService) OnExamineeConnect(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	if _, err := svc.q.SetSubmissionOnConnect(ctx, store.SetSubmissionOnConnectParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	}); err != nil {
		return err
	}

	return nil
}

func (svc *sessionService) OnExamineeDisconnect(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	if _, err := svc.q.SetSubmissionOnDisconnect(ctx, store.SetSubmissionOnDisconnectParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	return nil
}

// ------------------------------------------
// ------------------------------------------

func (svc *sessionService) OnGraceExpired(ctx context.Context, userID, sessionID uuid.UUID) error {

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != store.SessionStatusStarted {
		log.Printf(
			"Session:(%s) grace period expired for User:(%s). Session has not been started, no issue",
			sessionID,
			userID,
		)
		return nil
	}

	if _, err := svc.q.LogProctorEvent(ctx, store.LogProctorEventParams{
		SessionID:  sessionID,
		ExamineeID: userID,
		Type:       "EXTENDED_DISCONNECT",
		OccurredAt: time.Now(),
	}); err != nil {
		return err
	}

	// Only flag the user if in a disconnected state
	if _, err := svc.q.SetSubmissionFlaggedFromDisconnect(
		ctx,
		store.SetSubmissionFlaggedFromDisconnectParams{
			SessionID:  sessionID,
			ExamineeID: userID,
		},
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	// Send flagged event to specific client
	svc.hub.SendToUser(sessionID, userID, Message{
		Type: MessageTypeSessionFlagged,
	})

	svc.hub.RemoveMember(userID, sessionID)

	log.Printf(
		"Session:(%s), User:(%s) failed to reconnect. Flagged script",
		sessionID,
		userID,
	)

	return nil
}

func (svc *sessionService) SendStateSync(ctx context.Context, userID, sessionID uuid.UUID) error {

	session, err := svc.q.FindSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	svc.hub.SendToUser(sessionID, userID, Message{
		Type: MessageTypeSessionStateSync,
		Data: json.Wrapper{
			"status": session.Status,
		},
	})

	return nil
}

func (svc *sessionService) HandleProctorEvent(ctx context.Context, user store.User, event ProctorEvent) error {

	if err := common.RequireOwner(user, event.ExamineeID); err != nil {
		return err
	}

	// tx, err := svc.pool.Begin(ctx)
	// if err != nil {
	// 	return err
	// }
	// defer tx.Rollback(ctx)
	// qtx := svc.q.WithTx(tx)

	if _, err := svc.q.LogProctorEvent(ctx, store.LogProctorEventParams{
		SessionID:  event.SessionID,
		ExamineeID: event.ExamineeID,
		Type:       event.Type,
		OccurredAt: event.OccurredAt,
	}); err != nil {
		return err
	}

	// For now just flag every event type

	if _, err := svc.q.SetSubmissionFlaggedFromEditable(ctx, store.SetSubmissionFlaggedFromEditableParams{
		SessionID:  event.SessionID,
		ExamineeID: event.ExamineeID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Session:(%s) Apparently not editable, check DB", event.SessionID)
			return nil
		}
		return err
	}

	// if err := tx.Commit(ctx); err != nil {
	// 	return err
	// }

	svc.hub.SendToUser(event.SessionID, event.ExamineeID, Message{
		Type: MessageTypeSessionFlagged,
	})

	return nil
}

// ------------------------------------------------------------------------------------
// --- Marking Services ---------------------------------------------------------------
// ------------------------------------------------------------------------------------

func (svc *sessionService) MarkSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
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
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
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

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return "", err
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

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return []Session{}, err
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

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return Session{}, err
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
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return Session{}, err
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
