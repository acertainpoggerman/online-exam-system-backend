package submissions

import (
	"context"
	"fmt"
	"log"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/core/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, user store.User, data CreateSubmissionBody) (Submission, error)
	FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error)
	UpdateSubAnswerForQuestion(ctx context.Context, user store.User, submissionID uuid.UUID, questionID uuid.UUID, answer Answer) error
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

func (svc *submissionService) FindSubmissionByID(ctx context.Context, user store.User, submissionID uuid.UUID) (Submission, error) {

	sub, err := svc.q.FindSubmissionByID(ctx, submissionID)
	if err != nil {
		return Submission{}, err
	}

	answerFields, err := svc.q.FindSubmissionAnswers(ctx, submissionID)
	if err != nil {
		return Submission{}, err
	}

	var answers []Answer
	for _, answer := range answerFields {
		answers = append(answers, Answer{
			QuestionID: answer.QuestionID,
			Value:      answer.Value,
		})
	}

	return Submission{Submission: sub, Answers: answers}, nil
}

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

func (svc *submissionService) UpdateSubAnswerForQuestion(ctx context.Context, user store.User, submissionID uuid.UUID, questionID uuid.UUID, answer Answer) error {

	if user.Role != store.UserRoleExaminee {
		return fmt.Errorf("User role: \"%s\" not allowed", user.Role)
	}

	log.Println(answer.Value)

	return svc.q.ReplaceSubAnswerForQuestion(ctx, store.ReplaceSubAnswerForQuestionParams{
		SubmissionID: submissionID,
		QuestionID:   questionID,
		Value:        answer.Value,
	})
}

func (svc *submissionService) MarkSubmission(ctx context.Context, qtx *store.Queries, submissionID uuid.UUID) error {

	answers, err := qtx.FindSubmissionAnswers(ctx, submissionID)
	if err != nil {
		return err
	}

	for _, answer := range answers {
		question, err := svc.script.FindQuestionByID(ctx, answer.QuestionID)
		if err != nil {
			return err
		}

		var mark int32
		switch question.SubQuestion.(type) {

		// ------------------------------------------------------

		case *scripts.ChoiceQuestion:
			subq := question.SubQuestion.(*scripts.ChoiceQuestion)
			if svc.isCorrectChoiceQuestion(
				subq.IsMultipleChoice,
				question.AnswerKey,
				answer.Value,
			) {
				mark = 1
			}

		// ------------------------------------------------------

		case *scripts.TextQuestion:
			if svc.isCorrectTextQuestion(
				question.AnswerKey,
				answer.Value,
			) {
				mark = 1
			}

		// ------------------------------------------------------

		default:
			return fmt.Errorf("Invalid subquestion type: \"%T\"", question.SubQuestion)
		}

		if _, err := qtx.SetSubmissionQuestionMark(ctx, store.SetSubmissionQuestionMarkParams{
			SubmissionID: submissionID,
			QuestionID:   answer.QuestionID,
			Mark:         mark,
		}); err != nil {
			return err
		}
	}

	if _, err := qtx.CalculateSubmisionMark(ctx, submissionID); err != nil {
		return err
	}

	return nil
}

func (svc *submissionService) isCorrectChoiceQuestion(isMultipleChoice bool, answerKey []string, response []string) bool {

	if len(answerKey) == 0 {
		return true // Signal questions without answer keys as correct for now
	}
	if len(response) < 1 {
		return false // No answer given, mark incorrect
	}

	if isMultipleChoice {
		if len(response) != 1 {
			return false // Multiple answers given for a mcq, mark incorrect
		}
		return slices.Contains(answerKey, response[0]) // Will mark correct if answer matches a key
	}

	if common.EqualUnordered(answerKey, response) {
		return true // Response matches answer key, mark correct
	}
	return false // Response did not match answer key, mark incorrect
}

func (svc *submissionService) isCorrectTextQuestion(answerKey []string, response []string) bool {

	if len(answerKey) == 0 {
		return true // Signal questions without answer keys as correct for now
	}
	if len(response) != 1 {
		return false // Incorrect count of answers given (should be 1), mark incorrect
	}

	return slices.Contains(answerKey, response[0]) // Will mark correct if answer matches a key
}
