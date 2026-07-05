package submissions

import (
	"context"
	"fmt"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/core/sessions"
	"github.com/acertainpoggerman/online-exam-system/internal/core/users"
	"github.com/acertainpoggerman/online-exam-system/internal/update"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (string, error)
	FindSubmissions(ctx context.Context, user store.User) ([]Submission, error)
	FindSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) ([]Submission, error)
	FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error)
	UpdateSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID, includeSubmission bool, actions []update.UpdateAction) (*Submission, error)
}

type submissionService struct {
	q    *store.Queries
	pool *pgxpool.Pool

	script  scripts.ExtScriptService
	user    users.ExtUserService
	session sessions.ExtSessionService
}

func NewSubmissionService(
	q *store.Queries,
	pool *pgxpool.Pool,
	script scripts.ExtScriptService,
	user users.ExtUserService,
	session sessions.ExtSessionService,
) SubmissionService {
	return &submissionService{q, pool, script, user, session}
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

func (svc *submissionService) FindSubmissionsForSession(ctx context.Context, user store.User, sessionID uuid.UUID) ([]Submission, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return []Submission{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// --- Determine the role of the user -----------------------------------------------------------
	// |-> Examiner : Check if they own the session, then get all submissions associated with session
	// |-> Examinee : 403 Forbidden (Examinee does not own sessions)
	// ----------------------------------------------------------------------------------------------

	if user.Role != store.UserRoleExaminer {
		return []Submission{}, fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	if session, err := svc.session.FindSessionByID(ctx, user, sessionID); err != nil {
		return []Submission{}, fmt.Errorf("Cannot find session with ID: %s", sessionID)
	} else if session.CreatorID != user.ID {
		return []Submission{}, fmt.Errorf("Cannot access session resources")
	}

	submissions, err := qtx.FindSubmissionsForSession(ctx, sessionID)
	if err != nil {
		return []Submission{}, err
	}

	return common.Map(submissions, func(sub store.Submission) Submission {
		return Submission{Submission: sub}
	}), tx.Commit(ctx)
}

func (svc *submissionService) FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Submission{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	submission, err := qtx.FindSubmissionByID(ctx, submissionID)
	if err != nil {
		return Submission{}, err
	}

	switch user.Role {

	// ----------------------------------------------------------------------------
	// --- Role Checking [User.Examinee] ------------------------------------------
	// ----------------------------------------------------------------------------

	case store.UserRoleExaminee:

		if submission.ExamineeID != user.ID {
			return Submission{}, fmt.Errorf("Examinee does not own this submission")
		}

		details, err := qtx.FindAnswersForSubmission(ctx, submissionID)
		if err != nil {
			return Submission{}, err
		}

		answers := make([]Answer, len(details))
		for idx, detail := range details {

			// |-> Adding Questions to Submission if graded

			answers[idx] = Answer{}
			value, err := qtx.FindAnswerValuesForQuestion(ctx, store.FindAnswerValuesForQuestionParams{
				SubmissionID: detail.SubmissionID,
				QuestionID:   detail.QuestionID,
			})
			if err != nil {
				return Submission{}, err
			}
			answers[idx].Value = value

			// |-> Adding Questions to Submission if graded

			if submission.Status == store.SubmissionStatusGraded {
				question, err := svc.script.FindQuestionByID(ctx, detail.QuestionID)
				if err != nil {
					return Submission{}, err
				}
				answers[idx].Question = question
			}

		}

		return Submission{
			Submission: submission,
			Answers:    answers,
		}, tx.Commit(ctx)

	// ----------------------------------------------------------------------------
	// --- Role Checking [User.Examiner] ------------------------------------------
	// ----------------------------------------------------------------------------

	// [X] Check if the submission is not mutable
	// [ ] Check if the submission is for a session owned by User.Examiner

	case store.UserRoleExaminer:

		if submission.Status == store.SubmissionStatusMutable {
			return Submission{}, fmt.Errorf("Cannot access resource")
		}

		details, err := qtx.FindAnswersForSubmission(ctx, submissionID)
		if err != nil {
			return Submission{}, err
		}

		examinee, err := svc.user.FindExtUserByID(ctx, submission.ExamineeID)
		if err != nil {
			return Submission{}, err
		}

		answers := make([]Answer, len(details))
		for idx, detail := range details {

			answers[idx] = Answer{}
			value, err := qtx.FindAnswerValuesForQuestion(ctx, store.FindAnswerValuesForQuestionParams{
				SubmissionID: detail.SubmissionID,
				QuestionID:   detail.QuestionID,
			})
			if err != nil {
				return Submission{}, err
			}
			answers[idx].Value = value

			question, err := svc.script.FindQuestionByID(ctx, detail.QuestionID)
			if err != nil {
				return Submission{}, err
			}
			answers[idx].Question = question

		}

		return Submission{
			Submission: submission,
			Examinee:   examinee,
			Answers:    answers,
		}, tx.Commit(ctx)

	// ----------------------------------------------------------------------------

	default:
		return Submission{}, fmt.Errorf("Cannot access resource")
	}
}

func (svc *submissionService) CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (string, error) {

	if user.Role != store.UserRoleExaminee {
		return "", fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	qtx := svc.q.WithTx(tx)
	id, err := qtx.CreateSubmission(ctx, store.CreateSubmissionParams{
		SessionID:  data.SessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		return "", err
	}

	return id.String(), tx.Commit(ctx)
}

func (svc *submissionService) UpdateSubmissionByID(
	ctx context.Context,
	user store.User,
	submissionID uuid.UUID,
	includeSubmission bool,
	actions []update.UpdateAction,
) (*Submission, error) {

	if len(actions) == 0 {
		return nil, fmt.Errorf("No actions provided")
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	qtx := svc.q.WithTx(tx)

	// Only allowing examinees that own submission to modify

	if user.Role != store.UserRoleExaminee {
		return nil, fmt.Errorf("Cannot access resource")
	}

	if submission, err := qtx.FindSubmissionByID(ctx, submissionID); err != nil {
		return nil, err
	} else if submission.ExamineeID != user.ID {
		return nil, fmt.Errorf("Cannot access resource, not owned")
	}

	// Handling "/grade" actions (op:replace only)

	// if err := update.ExecuteLastInstanceOfAction(
	// 	actions,
	// 	update.OperationReplace,
	// 	"/grade",
	// 	svc.updateSubmissionGrade(ctx, qtx, submissionID),
	// ); err != nil {
	// 	return nil, err
	// }

	// Handling "/answers/{question_id}" actions (op:replace only)

	route := update.RoutingMap{
		"/answers/{question_id}": {update.OperationReplace: svc.updateSubmissionAnswer(ctx, qtx, submissionID)},
	}

	for _, action := range actions {
		if err := update.ExecuteAction(action, route); err != nil {
			return nil, err
		}
	}

	// Returning submission

	if includeSubmission {
		submission, err := svc.FindSubmissionByID(ctx, user, submissionID)
		if err != nil {
			return nil, err
		}
		return &submission, tx.Commit(ctx)
	}

	return nil, tx.Commit(ctx)
}

// -----------------------------------------------------------------------
// --- UpdateSubmission Action Handlers ----------------------------------
// -----------------------------------------------------------------------

func (svc *submissionService) updateSubmissionAnswer(
	ctx context.Context,
	qtx *store.Queries,
	submissionID uuid.UUID,
) update.UpdateActionHandler {

	return func(action update.UpdateAction, params map[string]string) error {

		questionID, ok := params["question_id"]
		if !ok {
			return fmt.Errorf("Question ID for answer not provided")
		}

		vs, ok := action.Value.([]any)
		if !ok {
			return fmt.Errorf("Value passed: \"%v\" is not a list", action.Value)
		}

		answers := common.Map(vs, func(v any) string {
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		})

		if !ok {
			return fmt.Errorf("Invalid answer: \"%v\" provided, expecting list of strings", action.Value)
		}

		parsedQuestionID, err := uuid.Parse(questionID)
		if err != nil {
			return err
		}

		if len(answers) == 0 {
			_, err := qtx.DeleteAnswerValuesByIDs(ctx, store.DeleteAnswerValuesByIDsParams{
				SubmissionID: submissionID,
				QuestionID:   parsedQuestionID,
			})
			return err // Early break because there's no need to do more beyond clearing all answers given.
		}

		qtype, err := qtx.FindQuestionTypeByID(ctx, parsedQuestionID)
		if err != nil {
			return err
		}

		switch qtype {
		case store.QuestionTypeText:

			if answerLength := len(answers); answerLength > 1 {
				return fmt.Errorf(
					"Too many answers provided for question of type: %v, (%v provided, requires 1 only)",
					qtype, answerLength,
				)
			}
			answer := answers[0]

			if _, err := qtx.DeleteAnswerValuesByIDs(ctx, store.DeleteAnswerValuesByIDsParams{
				SubmissionID: submissionID,
				QuestionID:   parsedQuestionID,
			}); err != nil {
				return err
			}

			if err := qtx.CreateAnswerValue(ctx, store.CreateAnswerValueParams{
				SubmissionID: submissionID,
				QuestionID:   parsedQuestionID,
				Value:        answer,
			}); err != nil {
				return err
			}

		case store.QuestionTypeChoice:

			q, err := qtx.FindChoiceQuestionByID(ctx, parsedQuestionID)
			if err != nil {
				return err
			}

			is_mcq := q.IsMultipleChoice

			if is_mcq {

				if answerLength := len(answers); answerLength > 1 {
					return fmt.Errorf(
						"Too many answers provided for question of type: %v, (%v provided, requires 1 only)",
						qtype, answerLength,
					)
				}
				answer := answers[0]

				if _, err := qtx.DeleteAnswerValuesByIDs(ctx, store.DeleteAnswerValuesByIDsParams{
					SubmissionID: submissionID,
					QuestionID:   parsedQuestionID,
				}); err != nil {
					return err
				}

				if err := qtx.CreateAnswerValue(ctx, store.CreateAnswerValueParams{
					SubmissionID: submissionID,
					QuestionID:   parsedQuestionID,
					Value:        answer,
				}); err != nil {
					return err
				}

			} else {

				count, err := qtx.FindOptionCountForQuestion(ctx, parsedQuestionID)
				if err != nil {
					return err
				}

				if answerLength := len(answers); answerLength > int(count) {
					return fmt.Errorf(
						"Too many answers provided for question of type: %v, (%v provided, maximum of %v allowed)",
						qtype, answerLength, count,
					)
				}

				if _, err := qtx.DeleteAnswerValuesByIDs(ctx, store.DeleteAnswerValuesByIDsParams{
					SubmissionID: submissionID,
					QuestionID:   parsedQuestionID,
				}); err != nil {
					return err
				}

				if _, err := qtx.CreateAnswerValues(ctx, common.Map(answers, func(a string) store.CreateAnswerValuesParams {
					return store.CreateAnswerValuesParams{
						SubmissionID: submissionID,
						QuestionID:   parsedQuestionID,
						Value:        a,
					}
				})); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func (svc *submissionService) updateSubmissionGrade(
	ctx context.Context,
	qtx *store.Queries,
	submissionID uuid.UUID,
) update.UpdateActionHandler {

	return func(action update.UpdateAction, params map[string]string) error {

		newGrade, ok := action.Value.(*int32)
		if !ok {
			return fmt.Errorf("Invalid grade: %v", action.Value)
		}

		return qtx.UpdateSubmissionGrade(ctx, store.UpdateSubmissionGradeParams{
			ID:    submissionID,
			Grade: newGrade,
		})
	}
}
