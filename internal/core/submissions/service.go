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
