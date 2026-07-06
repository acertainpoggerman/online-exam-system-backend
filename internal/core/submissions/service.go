package submissions

import (
	"context"
	"fmt"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/core/users"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (Submission, error)
	FindSubmissions(ctx context.Context, user store.User) ([]Submission, error)
	// FindSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) ([]Submission, error)
	// FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error)
}

type ExtSubmissionService interface {
	CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (Submission, error)
}

type submissionService struct {
	q    *store.Queries
	pool *pgxpool.Pool

	script scripts.ExtScriptService
	user   users.ExtUserService
}

func NewSubmissionService(
	q *store.Queries,
	pool *pgxpool.Pool,
	script scripts.ExtScriptService,
	user users.ExtUserService,
) *submissionService {
	return &submissionService{q, pool, script, user}
}

func (svc *submissionService) FindSubmissions(ctx context.Context, user store.User) ([]Submission, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return []Submission{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// Determine the role of the user
	// |-> Examiner : Check if they own the session, then get all submissions associated with session
	// |-> Examinee : 403 Forbidden (Examinee does not own submissions)

	if user.Role != store.UserRoleExaminee {
		return []Submission{}, fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	subs, err := qtx.FindSubmissionsForExaminee(ctx, user.ID)
	if err != nil {
		return []Submission{}, err
	}

	submissions := common.Map(subs, func(sub store.Submission) Submission {
		return Submission{Submission: sub}
	})

	return submissions, tx.Commit(ctx)
}

// func (svc *submissionService) FindSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) ([]Submission, error) {

// 	tx, err := svc.pool.Begin(ctx)
// 	if err != nil {
// 		return []Submission{}, err
// 	}
// 	defer tx.Rollback(ctx)
// 	qtx := svc.q.WithTx(tx)

// 	// --- Determine the role of the user -----------------------------------------------------------
// 	// |-> Examiner : Check if they own the session, then get all submissions associated with session
// 	// |-> Examinee : 403 Forbidden (Examinee does not own sessions)
// 	// ----------------------------------------------------------------------------------------------

// 	if user.Role != store.UserRoleExaminer {
// 		return []Submission{}, fmt.Errorf("User role: \"%s\" not allowed", user.Role)
// 	}

// 	if session, err := svc.session.FindSessionByID(ctx, user, sessionID); err != nil {
// 		return []Submission{}, fmt.Errorf("Cannot find session with ID: %s", sessionID)
// 	} else if session.CreatorID != user.ID {
// 		return []Submission{}, fmt.Errorf("Cannot access session resources")
// 	}

// 	submissions, err := qtx.FindSubmissionsForSession(ctx, sessionID)
// 	if err != nil {
// 		return []Submission{}, err
// 	}

// 	return common.Map(submissions, func(sub store.Submission) Submission {
// 		return Submission{Submission: sub}
// 	}), tx.Commit(ctx)
// }

// func (svc *submissionService) FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error) {

// 	tx, err := svc.pool.Begin(ctx)
// 	if err != nil {
// 		return Submission{}, err
// 	}
// 	defer tx.Rollback(ctx)
// 	qtx := svc.q.WithTx(tx)

// 	submission, err := qtx.FindSubmissionByID(ctx, submissionID)
// 	if err != nil {
// 		return Submission{}, err
// 	}

// 	switch user.Role {

// 	// ----------------------------------------------------------------------------
// 	// --- Role Checking [User.Examinee] ------------------------------------------
// 	// ----------------------------------------------------------------------------

// 	case store.UserRoleExaminee:

// 		if submission.ExamineeID != user.ID {
// 			return Submission{}, fmt.Errorf("Examinee does not own this submission")
// 		}

// 		details, err := qtx.FindAnswersForSubmission(ctx, submissionID)
// 		if err != nil {
// 			return Submission{}, err
// 		}

// 		answers := make([]Answer, len(details))
// 		for idx, detail := range details {

// 			// |-> Adding Questions to Submission if graded

// 			answers[idx] = Answer{}
// 			value, err := qtx.FindAnswerValuesForQuestion(ctx, store.FindAnswerValuesForQuestionParams{
// 				SubmissionID: detail.SubmissionID,
// 				QuestionID:   detail.QuestionID,
// 			})
// 			if err != nil {
// 				return Submission{}, err
// 			}
// 			answers[idx].Value = value

// 			// |-> Adding Questions to Submission if graded

// 			if submission.Status == store.SubmissionStatusGraded {
// 				question, err := svc.script.FindQuestionByID(ctx, detail.QuestionID)
// 				if err != nil {
// 					return Submission{}, err
// 				}
// 				answers[idx].Question = question
// 			}

// 		}

// 		return Submission{
// 			Submission: submission,
// 			Answers:    answers,
// 		}, tx.Commit(ctx)

// 	// ----------------------------------------------------------------------------
// 	// --- Role Checking [User.Examiner] ------------------------------------------
// 	// ----------------------------------------------------------------------------

// 	// [X] Check if the submission is not mutable
// 	// [ ] Check if the submission is for a session owned by User.Examiner

// 	case store.UserRoleExaminer:

// 		if submission.Status == store.SubmissionStatusMutable {
// 			return Submission{}, fmt.Errorf("Cannot access resource")
// 		}

// 		details, err := qtx.FindAnswersForSubmission(ctx, submissionID)
// 		if err != nil {
// 			return Submission{}, err
// 		}

// 		examinee, err := svc.user.FindExtUserByID(ctx, submission.ExamineeID)
// 		if err != nil {
// 			return Submission{}, err
// 		}

// 		answers := make([]Answer, len(details))
// 		for idx, detail := range details {

// 			answers[idx] = Answer{}
// 			value, err := qtx.FindAnswerValuesForQuestion(ctx, store.FindAnswerValuesForQuestionParams{
// 				SubmissionID: detail.SubmissionID,
// 				QuestionID:   detail.QuestionID,
// 			})
// 			if err != nil {
// 				return Submission{}, err
// 			}
// 			answers[idx].Value = value

// 			question, err := svc.script.FindQuestionByID(ctx, detail.QuestionID)
// 			if err != nil {
// 				return Submission{}, err
// 			}
// 			answers[idx].Question = question

// 		}

// 		return Submission{
// 			Submission: submission,
// 			Examinee:   examinee,
// 			Answers:    answers,
// 		}, tx.Commit(ctx)

// 	// ----------------------------------------------------------------------------

// 	default:
// 		return Submission{}, fmt.Errorf("Cannot access resource")
// 	}
// }

func (svc *submissionService) CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (Submission, error) {

	if user.Role != store.UserRoleExaminee {
		return Submission{}, fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	sub, err := svc.q.CreateSubmission(ctx, store.CreateSubmissionParams{
		SessionID:  data.SessionID,
		ExamineeID: user.ID,
	})

	return Submission{Submission: sub}, err
}
