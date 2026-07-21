package examscripts

import (
	"context"
	"fmt"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/google/uuid"
)

// Creates a Question for a script given the scriptID
// and an instance of the question struct.
func (svc *scriptService) CreateQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, question Question) (Question, error) {

	// ------------------------------------------------------------
	// --- Check that User is Examiner ----------------------------

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return Question{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ------------------------------------------------------------
	// --- Ensuing Examiner owns the Script -----------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if err := common.RequireOwner(user, script.CreatorID); err != nil {
		return Question{}, err
	}

	// ------------------------------------------------------------
	// --- Creating the Question ----------------------------------

	var questionID uuid.UUID

	switch subq := question.SubQuestion.(type) {

	// ------------------------------------------------------------

	case *ChoiceQuestion:
		questionID, err = qtx.CreateChoiceQuestion(ctx, store.CreateChoiceQuestionParams{
			ScriptID:         scriptID,
			Text:             question.Text,
			ImageUrl:         question.ImageUrl,
			Mark:             question.Mark,
			IsMultipleChoice: subq.IsMultipleChoice,
		})
		if err != nil {
			return Question{}, err
		}

		var values []string
		var imgUrls []string

		for _, option := range subq.Options {
			values = append(values, option.Value)
			imgUrls = append(imgUrls, common.PtrToString(option.ImageUrl))
		}

		if err := qtx.ReplaceOptionsForQuestion(ctx, store.ReplaceOptionsForQuestionParams{
			QuestionID: questionID,
			Values:     values,
			ImageUrls:  imgUrls,
		}); err != nil {
			return Question{}, err
		}

	// ------------------------------------------------------------

	case *TextQuestion:
		questionID, err = qtx.CreateTextQuestion(ctx, store.CreateTextQuestionParams{
			ScriptID:    scriptID,
			Text:        question.Text,
			ImageUrl:    question.ImageUrl,
			Mark:        question.Mark,
			IsShortText: subq.IsShortText,
		})
		if err != nil {
			return Question{}, err
		}

	// ------------------------------------------------------------

	default:
		return Question{}, fmt.Errorf("Question Type: \"%T\" does not exist", question.SubQuestion)
	}

	// ------------------------------------------------------------
	// --- Replacing the Question's Answer Key --------------------

	if err := qtx.ReplaceAnswerKeyForQuestion(ctx, store.ReplaceAnswerKeyForQuestionParams{
		QuestionID: questionID,
		AnswerKey:  question.AnswerKey,
	}); err != nil {
		return Question{}, err
	}

	created, err := svc.findQuestionByID(ctx, qtx, questionID, true)
	if err != nil {
		return Question{}, err
	}

	return created, tx.Commit(ctx)
}

// Replaces a question for a script given the scriptID
// and the questionID with the instance of the question.
func (svc *scriptService) ReplaceQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID, question Question) (Question, error) {

	// ------------------------------------------------------------
	// --- Check that User is Examiner ----------------------------

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return Question{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ------------------------------------------------------------
	// --- Ensuing Examiner owns the Script -----------------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if err := common.RequireOwner(user, script.CreatorID); err != nil {
		return Question{}, err
	}

	// ------------------------------------------------------------
	// --- Add New Question Sub-type Details ----------------------

	switch subq := question.SubQuestion.(type) {

	// ------------------------------------------------------------

	case *ChoiceQuestion:
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
			imgUrls = append(imgUrls, common.PtrToString(option.ImageUrl))
		}

		if err := qtx.ReplaceOptionsForQuestion(ctx, store.ReplaceOptionsForQuestionParams{
			QuestionID: questionID,
			Values:     values,
			ImageUrls:  imgUrls,
		}); err != nil {
			return Question{}, err
		}

	// ------------------------------------------------------------

	case *TextQuestion:
		if err = qtx.UpsertTextQuestion(ctx, store.UpsertTextQuestionParams{
			ID:          questionID,
			IsShortText: subq.IsShortText,
		}); err != nil {
			return Question{}, err
		}

	// ------------------------------------------------------------

	default:
		return Question{}, fmt.Errorf("Question Type: \"%T\" does not exist", question.SubQuestion)
	}

	// ------------------------------------------------------------
	// --- Update Super-type Details ------------------------------

	if err := qtx.UpdateQuestionFields(ctx, store.UpdateQuestionFieldsParams{
		ID:       questionID,
		Text:     question.Text,
		ImageUrl: question.ImageUrl,
		Mark:     question.Mark,
	}); err != nil {
		return Question{}, err
	}

	if err := qtx.ReplaceAnswerKeyForQuestion(ctx, store.ReplaceAnswerKeyForQuestionParams{
		QuestionID: questionID,
		AnswerKey:  question.AnswerKey,
	}); err != nil {
		return Question{}, err
	}

	// ------------------------------------------------------------
	// --- Get Full Question --------------------------------------

	replaced, err := svc.findQuestionByID(ctx, qtx, questionID, true)
	if err != nil {
		return Question{}, err
	}

	return replaced, tx.Commit(ctx)
}

// Deletes a question for a script given the scriptID and
// the questionID.
func (svc *scriptService) DeleteQuestion(ctx context.Context, user store.User, scriptID uuid.UUID, questionID uuid.UUID) (Question, error) {

	// ------------------------------------------------------------
	// --- Check that User is Examiner ----------------------------

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return Question{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return Question{}, err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// ------------------------------------------------------------
	// --- Ensuing User (as Examiner) owns the Script -------------

	if script, err := qtx.FindScriptByID(ctx, scriptID); err != nil {
		return Question{}, err
	} else if user.ID != script.CreatorID {
		return Question{}, fmt.Errorf("Cannot modify resource")
	}

	// ------------------------------------------------------------
	// --- Get Full Question & Delete it --------------------------

	deleted, err := svc.findQuestionByID(ctx, qtx, questionID, true)
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

// -----------------------------------------------------------------------
// --- Implementing QuestionService Interface ----------------------------
// -----------------------------------------------------------------------

// Finds a question given its ID. getAnswerKey determines if the method
// should also return the question with its answer key.
func (svc *scriptService) FindQuestionByID(ctx context.Context, questionID uuid.UUID, getAnswerKey bool) (Question, error) {

	question, err := svc.findQuestionByID(ctx, svc.q, questionID, getAnswerKey)
	if err != nil {
		return Question{}, err
	}

	return question, nil
}

func (svc *scriptService) findQuestionByID(ctx context.Context, qtx *store.Queries, questionID uuid.UUID, getAnswerKey bool) (Question, error) {

	question, err := qtx.FindQuestionByID(ctx, questionID)
	if err != nil {
		return Question{}, err
	}

	switch question.Type {

	// ------------------------------------------------------------

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

		if getAnswerKey {
			answerKey, err := qtx.FindAnswerKeyForQuestion(ctx, question.ID)
			if err != nil {
				return Question{}, err
			}
			if len(answerKey) > 0 {
				final.AnswerKey = answerKey
			}
		}

		return final, nil

	// ------------------------------------------------------------

	case store.QuestionTypeText:
		subq, err := qtx.FindTextQuestionByID(ctx, question.ID)
		if err != nil {
			return Question{}, err
		}

		final := Question{
			Question:    question,
			SubQuestion: &TextQuestion{subq},
		}

		if getAnswerKey {
			answerKey, err := qtx.FindAnswerKeyForQuestion(ctx, question.ID)
			if err != nil {
				return Question{}, err
			}
			if len(answerKey) > 0 {
				final.AnswerKey = answerKey
			}
		}

		return final, nil

	// ------------------------------------------------------------

	default:
		return Question{}, fmt.Errorf("Question Type: \"%s\" does not exist", question.Type)
	}
}
