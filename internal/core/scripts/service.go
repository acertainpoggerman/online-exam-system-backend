package scripts

import (
	"context"
	"fmt"
	"time"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/api"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScriptService interface {
	CreateScript(ctx context.Context, user store.User, data CreateScriptBody) (Script, error)
	FindScripts(ctx context.Context, user store.User, cursor *api.Cursor, size int32, search string) ([]Script, int64, error)
	FindScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID) (Script, error)
	UpdateScriptFields(ctx context.Context, user store.User, scriptID uuid.UUID, data UpdateScriptBody) error
	DeleteScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID) error

	CreateQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, question Question) (Question, error)
	ReplaceQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID, question Question) (Question, error)
	DeleteQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID) (Question, error)

	FindScriptForSubmission(ctx context.Context, user store.User, submissionID uuid.UUID) (Script, error)
}

type ExtScriptService interface {
	FindQuestionByID(ctx context.Context, questionID uuid.UUID) (Question, error)
}

type scriptService struct {
	q    *store.Queries
	pool *pgxpool.Pool
}

func NewScriptService(q *store.Queries, pool *pgxpool.Pool) *scriptService {
	return &scriptService{q, pool}
}

// -----------------------------------------------------------------------
// --- Implementing ScriptService Interface ------------------------------
// -----------------------------------------------------------------------

func (svc *scriptService) FindScripts(ctx context.Context, user store.User, cursor *api.Cursor, size int32, search string) ([]Script, int64, error) {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return []Script{}, 0, err
	}

	if cursor == nil {
		cursor = &api.Cursor{Ts: time.Now(), ID: uuid.Max}
	}

	var scripts []store.Script
	var count int64
	var err error

	if scripts, err = svc.q.FindScriptsForExaminer(ctx, store.FindScriptsForExaminerParams{
		ExaminerID: user.ID,
		Search:     search,
		CursorTs:   cursor.Ts,
		CursorID:   cursor.ID,
		PageSize:   size,
	}); err != nil {
		return []Script{}, 0, err
	}

	if count, err = svc.q.FindScriptCountForExaminer(ctx, store.FindScriptCountForExaminerParams{
		ExaminerID: user.ID,
		Search:     search,
	}); err != nil {
		return []Script{}, 0, err
	}

	return common.Map(scripts, func(s store.Script) Script {
		return Script{Script: s}
	}), count, nil
}

func (svc *scriptService) FindScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID) (Script, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Script{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	script, err := qtx.FindScriptByID(ctx, scriptID)
	if err != nil {
		return Script{}, err
	}

	questions, err := qtx.FindQuestionsForScript(ctx, scriptID)
	if err != nil {
		return Script{}, err
	}

	qs := make([]Question, len(questions))
	for idx, question := range questions {

		answerKey, err := qtx.FindAnswerKeyForQuestion(ctx, question.ID)
		if err != nil {
			return Script{}, err
		}

		switch question.Type {

		// ---------------------------------------------------------------

		case store.QuestionTypeChoice:
			subq, err := qtx.FindChoiceQuestionByID(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			options, err := qtx.FindOptionsForQuestion(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			qs[idx] = Question{
				Question: question,
				SubQuestion: &ChoiceQuestion{
					ChoiceQuestion: subq,
					Options:        options,
				},
			}
			if len(answerKey) > 0 {
				qs[idx].AnswerKey = answerKey
			}

		// ---------------------------------------------------------------

		case store.QuestionTypeText:
			subq, err := qtx.FindTextQuestionByID(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			qs[idx] = Question{
				Question:    question,
				SubQuestion: &TextQuestion{subq},
			}
			if len(answerKey) > 0 {
				qs[idx].AnswerKey = answerKey
			}

		// ---------------------------------------------------------------

		default:
			return Script{}, fmt.Errorf("Question Type: \"%s\" does not exist", question.Type)
		}
	}

	return Script{
		Script:    script,
		Questions: qs,
	}, tx.Commit(ctx)
}

func (svc *scriptService) CreateScript(ctx context.Context, user store.User, data CreateScriptBody) (Script, error) {

	if user.Role != store.UserRoleExaminer {
		return Script{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Script{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	script, err := qtx.CreateScript(ctx, store.CreateScriptParams{
		Title:       data.Title,
		Heading:     data.Heading,
		Description: data.Description,
		CreatorID:   user.ID,
	})
	if err != nil {
		return Script{}, err
	}

	return Script{Script: script}, tx.Commit(ctx)
}

func (svc *scriptService) UpdateScriptFields(ctx context.Context, user store.User, scriptID uuid.UUID, data UpdateScriptBody) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuing User (as Examiner) owns the Script -----------------------------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return err
	} else if user.ID != script.CreatorID {
		return fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Replacing Script Details -----------------------------------------------------------

	if _, err := qtx.UpdateScriptFields(ctx, store.UpdateScriptFieldsParams{
		ID:          scriptID,
		Title:       data.Title,
		Heading:     data.Heading,
		Description: data.Description,
		DefaultMark: int32(data.DefaultMark),
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (svc *scriptService) DeleteScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID) error {

	if user.Role != store.UserRoleExaminer {
		return fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuing User (as Examiner) owns the Script -----------------------------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return err
	} else if user.ID != script.CreatorID {
		return fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Deleting the Script ----------------------------------------------------------------

	if err := qtx.DeleteScript(ctx, scriptID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (svc *scriptService) FindScriptForSubmission(ctx context.Context, user store.User, submissionID uuid.UUID) (Script, error) {

	if user.Role != store.UserRoleExaminee {
		return Script{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Script{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	script, err := qtx.FindScriptForSubmission(ctx, submissionID)
	if err != nil {
		return Script{}, err
	}

	questions, err := qtx.FindQuestionsForSubmission(ctx, submissionID)
	if err != nil {
		return Script{}, err
	}

	qs := make([]Question, len(questions))
	for idx, question := range questions {

		var answerKey []string
		// answerKey, err = qtx.FindAnswerKeyForQuestion(ctx, question.ID)
		// if err != nil {
		// 	return Script{}, err
		// }

		switch question.Type {

		// ---------------------------------------------------------------

		case store.QuestionTypeChoice:
			subq, err := qtx.FindChoiceQuestionByID(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			options, err := qtx.FindOptionsForQuestionShuffled(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			qs[idx] = Question{
				Question: question,
				SubQuestion: &ChoiceQuestion{
					ChoiceQuestion: subq,
					Options:        options,
				},
			}
			if len(answerKey) > 0 {
				qs[idx].AnswerKey = answerKey
			}

		// ---------------------------------------------------------------

		case store.QuestionTypeText:
			subq, err := qtx.FindTextQuestionByID(ctx, question.ID)
			if err != nil {
				return Script{}, err
			}
			qs[idx] = Question{
				Question:    question,
				SubQuestion: &TextQuestion{subq},
			}
			if len(answerKey) > 0 {
				qs[idx].AnswerKey = answerKey
			}

		// ---------------------------------------------------------------

		default:
			return Script{}, fmt.Errorf("Question Type: \"%s\" does not exist", question.Type)
		}
	}

	return Script{
		Script:    script,
		Questions: qs,
	}, tx.Commit(ctx)

}
