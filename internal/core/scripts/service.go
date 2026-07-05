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
	UpdateScriptFields(ctx context.Context, user store.User, scriptID uuid.UUID, data CreateScriptBody) error
	DeleteScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID) error
	// UpdateScriptByID(ctx context.Context, user store.User, scriptID uuid.UUID, includeScript bool, actions []update.UpdateAction) (*Script, error)

	CreateQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, question Question) (Question, error)
	ReplaceQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID, question Question) (Question, error)
	DeleteQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID) (Question, error)
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
// --- Implementing QuestionService Interface ----------------------------
// -----------------------------------------------------------------------

func (svc *scriptService) FindQuestionByID(ctx context.Context, questionID uuid.UUID) (Question, error) {

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	question, err := svc.findQuestionByID(ctx, qtx, questionID)
	if err != nil {
		return Question{}, err
	}

	return question, tx.Commit(ctx)
}

func (svc *scriptService) findQuestionByID(ctx context.Context, qtx *store.Queries, questionID uuid.UUID) (Question, error) {

	question, err := qtx.FindQuestionByID(ctx, questionID)
	if err != nil {
		return Question{}, err
	}

	switch question.Type {

	case store.QuestionTypeChoice:
		subq, err := qtx.FindChoiceQuestionByID(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}
		options, err := qtx.FindOptionsForQuestion(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}

		final := Question{
			Question: question,
			SubQuestion: &ChoiceQuestion{
				ChoiceQuestion: subq,
				Options:        options,
			},
		}

		answerKey, err := qtx.FindAnswerKeyForQuestion(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}
		if len(answerKey) > 0 {
			final.AnswerKey = answerKey
		}

		return final, nil

	case store.QuestionTypeText:
		subq, err := qtx.FindTextQuestionByID(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}

		final := Question{
			Question:    question,
			SubQuestion: &TextQuestion{subq},
		}

		answerKey, err := qtx.FindAnswerKeyForQuestion(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}
		if len(answerKey) > 0 {
			final.AnswerKey = answerKey
		}

		return final, nil

	default:
		return Question{}, fmt.Errorf("Question Type: \"%s\" does not exist", question.Type)
	}
}

// -----------------------------------------------------------------------
// --- Implementing ScriptService Interface ------------------------------
// -----------------------------------------------------------------------

func (svc *scriptService) FindScripts(ctx context.Context, user store.User, cursor *api.Cursor, size int32, search string) ([]Script, int64, error) {

	if user.Role != store.UserRoleExaminer {
		return []Script{}, 0, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	if cursor == nil {
		cursor = &api.Cursor{Ts: time.Now(), ID: uuid.Max}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return []Script{}, 0, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	var scripts []store.Script
	var count int64

	if search == "" {
		if scripts, err = qtx.FindScriptsForExaminer(ctx, store.FindScriptsForExaminerParams{
			ExaminerID: user.ID,
			CursorTs:   cursor.Ts,
			CursorID:   cursor.ID,
			PageSize:   size,
		}); err != nil {
			return []Script{}, 0, err
		}

		if count, err = qtx.FindScriptCountForExaminer(ctx, user.ID); err != nil {
			return []Script{}, 0, err
		}
	} else {
		if scripts, err = qtx.SearchScriptsForExaminer(ctx, store.SearchScriptsForExaminerParams{
			ExaminerID: user.ID,
			Search:     search,
			CursorTs:   cursor.Ts,
			CursorID:   cursor.ID,
			PageSize:   size,
		}); err != nil {
			return []Script{}, 0, err
		}

		if count, err = qtx.SearchScriptCountForExaminer(ctx, store.SearchScriptCountForExaminerParams{
			ExaminerID: user.ID,
			Search:     search,
		}); err != nil {
			return []Script{}, 0, err
		}
	}

	return common.Map(scripts, func(s store.Script) Script {
		return Script{Script: s}
	}), count, tx.Commit(ctx)
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

func (svc *scriptService) UpdateScriptFields(ctx context.Context, user store.User, scriptID uuid.UUID, data CreateScriptBody) error {

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

	if err := qtx.UpdateScriptFields(ctx, store.UpdateScriptFieldsParams{
		ID:          scriptID,
		Title:       data.Title,
		Heading:     data.Heading,
		Description: data.Description,
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

// -----------------------------------------------------------------------
// --- Question Services -------------------------------------------------
// -----------------------------------------------------------------------

func (svc *scriptService) CreateQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, question Question) (Question, error) {

	// ----------------------------------------------------------------------------------------
	// --- Check that User is Examiner --------------------------------------------------------

	if user.Role != store.UserRoleExaminer {
		return Question{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuing User (as Examiner) owns the Script -----------------------------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if user.ID != script.CreatorID {
		return Question{}, fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Creating the Question --------------------------------------------------------------

	var questionID uuid.UUID

	switch question.SubQuestion.(type) {

	// ---------------------------------------------------------

	case *ChoiceQuestion:
		subq := question.SubQuestion.(*ChoiceQuestion)
		questionID, err = qtx.CreateChoiceQuestion(ctx, store.CreateChoiceQuestionParams{
			ScriptID:         scriptID,
			Text:             question.Text,
			ImageUrl:         question.ImageUrl,
			IsMultipleChoice: subq.IsMultipleChoice,
		})
		if err != nil {
			return Question{}, err
		}

		var values []string
		var imgUrls []string

		for _, option := range subq.Options {
			values = append(values, option.Value)
			imgUrls = append(imgUrls, common.ZeroNilString(option.ImageUrl))
		}

		if err := qtx.ReplaceOptionsForQuestion(ctx, store.ReplaceOptionsForQuestionParams{
			QuestionID: questionID,
			Values:     values,
			ImageUrls:  imgUrls,
		}); err != nil {
			return Question{}, err
		}

	// ---------------------------------------------------------

	case *TextQuestion:
		subq := question.SubQuestion.(*TextQuestion)
		questionID, err = qtx.CreateTextQuestion(ctx, store.CreateTextQuestionParams{
			ScriptID:    scriptID,
			Text:        question.Text,
			ImageUrl:    question.ImageUrl,
			IsShortText: subq.IsShortText,
		})
		if err != nil {
			return Question{}, err
		}

	// ---------------------------------------------------------

	default:
		return Question{}, fmt.Errorf("Question Type: \"%T\" does not exist", question.SubQuestion)
	}

	// ----------------------------------------------------------------------------------------
	// --- Replacing the Question's Answer Key ------------------------------------------------

	if err := qtx.ReplaceAnswerKeyForQuestion(ctx, store.ReplaceAnswerKeyForQuestionParams{
		QuestionID: questionID,
		AnswerKey:  question.AnswerKey,
	}); err != nil {
		return Question{}, err
	}

	created, err := svc.findQuestionByID(ctx, qtx, questionID)
	if err != nil {
		return Question{}, err
	}

	return created, tx.Commit(ctx)
}

func (svc *scriptService) ReplaceQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID, question Question) (Question, error) {

	// ----------------------------------------------------------------------------------------
	// --- Check that User is Examiner --------------------------------------------------------

	if user.Role != store.UserRoleExaminer {
		return Question{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuring User (as Examiner) owns the Script ----------------------------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if user.ID != script.CreatorID {
		return Question{}, fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Add New Question Sub-type Details --------------------------------------------------

	switch question.SubQuestion.(type) {

	// ---------------------------------------------------------

	case *ChoiceQuestion:
		subq := question.SubQuestion.(*ChoiceQuestion)
		if err = qtx.UpsertChoiceQuestion(ctx, store.UpsertChoiceQuestionParams{
			ID:               questionID,
			IsMultipleChoice: subq.IsMultipleChoice,
		}); err != nil {
			return Question{}, err
		}

		var values []string
		var imgUrls []string

		for _, option := range subq.Options {
			values = append(values, option.Value)
			imgUrls = append(imgUrls, common.ZeroNilString(option.ImageUrl))
		}

		if err := qtx.ReplaceOptionsForQuestion(ctx, store.ReplaceOptionsForQuestionParams{
			QuestionID: questionID,
			Values:     values,
			ImageUrls:  imgUrls,
		}); err != nil {
			return Question{}, err
		}

	// ---------------------------------------------------------

	case *TextQuestion:
		subq := question.SubQuestion.(*TextQuestion)
		if err = qtx.UpsertTextQuestion(ctx, store.UpsertTextQuestionParams{
			ID:          questionID,
			IsShortText: subq.IsShortText,
		}); err != nil {
			return Question{}, err
		}

	// ---------------------------------------------------------

	default:
		return Question{}, fmt.Errorf("Question Type: \"%T\" does not exist", question.SubQuestion)
	}

	// ----------------------------------------------------------------------------------------
	// --- Update Super-type Details ----------------------------------------------------------

	if err := qtx.UpdateQuestionFields(ctx, store.UpdateQuestionFieldsParams{
		ID:       questionID,
		Text:     question.Text,
		ImageUrl: question.ImageUrl,
	}); err != nil {
		return Question{}, err
	}

	if err := qtx.ReplaceAnswerKeyForQuestion(ctx, store.ReplaceAnswerKeyForQuestionParams{
		QuestionID: questionID,
		AnswerKey:  question.AnswerKey,
	}); err != nil {
		return Question{}, err
	}

	// ----------------------------------------------------------------------------------------
	// --- Get Full Question ------------------------------------------------------------------

	replaced, err := svc.findQuestionByID(ctx, qtx, questionID)
	if err != nil {
		return Question{}, err
	}

	return replaced, tx.Commit(ctx)
}

func (svc *scriptService) DeleteQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID) (Question, error) {

	// ----------------------------------------------------------------------------------------
	// --- Check that User is Examiner --------------------------------------------------------

	if user.Role != store.UserRoleExaminer {
		return Question{}, fmt.Errorf("Cannot access resource as: %s", user.Role)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ----------------------------------------------------------------------------------------
	// --- Ensuing User (as Examiner) owns the Script -----------------------------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if user.ID != script.CreatorID {
		return Question{}, fmt.Errorf("Cannot modify resource")
	}

	// ----------------------------------------------------------------------------------------
	// --- Get Full Question & Delete it ------------------------------------------------------

	deleted, err := svc.findQuestionByID(ctx, qtx, questionID)
	if err != nil {
		return Question{}, err
	}

	if deleted.ScriptID != scriptID {
		return Question{}, fmt.Errorf("Question does not belong to script")
	}

	if err := qtx.DeleteQuestion(ctx, questionID); err != nil {
		return Question{}, err
	}

	return deleted, tx.Commit(ctx)
}
