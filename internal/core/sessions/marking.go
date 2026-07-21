package sessions

import (
	"context"
	"fmt"
	"log"
	"slices"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/acertainpoggerman/online-exam-system/internal/core/examscripts"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------
// --- Automatic Marking -------------------------------------------
// -----------------------------------------------------------------

// Automatically gives the mark for the entire session's responses.
func (svc *sessionService) AutoMarkSession(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// -------------------------------------------------------------
	// --- Ensuring User (as Examiner) owns the Script -------------

	if session, err := qtx.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
	}

	// -------------------------------------------------------------
	// --- Marking Responses ---------------------------------------

	rows, err := qtx.FindSessionResponses(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if err := svc.autoMarkResponse(ctx, qtx, row.Response.SessionID, row.Response.ExamineeID); err != nil {
			log.Printf("Error in AutoMarkSession: autoMarkResponse (%v)", err)
			return apperr.ErrInternal
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error in AutoMarkSession: autoMarkResponse (%v)", err)
		return apperr.ErrInternal
	}

	return nil
}

// Automatically gives a mark for the submission
func (svc *sessionService) autoMarkResponse(ctx context.Context, q *store.Queries, sessionID uuid.UUID, examineeID uuid.UUID) error {

	rows, err := q.FindQuestionResponses(ctx, store.FindQuestionResponsesParams{
		SessionID:  sessionID,
		ExamineeID: examineeID,
	})
	if err != nil {
		log.Printf("Error in autoMarkResponse: Find QuestionResponses (%v)", err)
		return err
	}

	for _, row := range rows {

		question, err := svc.script.FindQuestionByID(ctx, row.QuestionResponse.QuestionID, true, false)
		if err != nil {
			return err
		}

		if len(question.AnswerKey) == 0 { // Ignore questions without an answer key
			continue
		}

		correct, err := svc.isCorrect(question, row.Value)
		if err != nil {
			return err
		}

		if _, err := q.AutoMarkQuestionResponse(ctx, store.AutoMarkQuestionResponseParams{
			IsCorrect:  correct,
			SessionID:  sessionID,
			ExamineeID: examineeID,
			QuestionID: row.QuestionResponse.QuestionID,
		}); err != nil {
			log.Printf("Error in autoMarkResponse: (%v)", err)
			return err
		}
	}

	if _, err := q.SetResponseStatusPostAutoMark(
		ctx,
		store.SetResponseStatusPostAutoMarkParams{
			SessionID:  sessionID,
			ExamineeID: examineeID,
		},
	); err != nil {
		log.Printf("Error in autoMarkResponse: StatusSet (%v)", err)
		return err
	}

	return nil
}

// Returns if the question answer given is correct or not.
func (svc *sessionService) isCorrect(question examscripts.Question, answer []string) (bool, error) {

	switch subq := question.SubQuestion.(type) {

	// ------------------------------------------------------

	case *examscripts.ChoiceQuestion:
		return svc.isCorrectChoiceQuestion(
			subq.IsMultipleChoice,
			question.AnswerKey,
			answer,
		), nil

	// ------------------------------------------------------

	case *examscripts.TextQuestion:
		return svc.isCorrectTextQuestion(
			question.AnswerKey,
			answer,
		), nil

	// ------------------------------------------------------

	default:
		return false, fmt.Errorf("Invalid subquestion type: \"%T\"", question.SubQuestion)
	}
}

// Returns if the choice question answer is correct based
// on the question type, answer key and response given.
func (svc *sessionService) isCorrectChoiceQuestion(isMultipleChoice bool, answerKey []string, response []string) bool {

	if len(response) < 1 {
		return false // No answer given, mark incorrect
	}

	if isMultipleChoice {
		if len(response) != 1 {
			return false // Multiple answers given for a mcq, mark incorrect
		}
		return slices.Contains(answerKey, response[0]) // Will mark correct if answer matches a key
	}

	return common.EqualUnordered(answerKey, response) // Will mark correct if answer fully matches the key
}

// Returns if the text question answer is correct based
// on the answer key and response given.
func (svc *sessionService) isCorrectTextQuestion(answerKey []string, response []string) bool {

	if len(response) != 1 {
		return false // Incorrect count of answers given (should be 1), mark incorrect
	}

	return slices.Contains(answerKey, response[0]) // Will mark correct if answer matches a key
}

// -----------------------------------------------------------------
// --- Examiner Marking --------------------------------------------
// -----------------------------------------------------------------

// Marks the questions given by the examiner for
// an examinee's response.
func (svc *sessionService) ExaminerMarkResponse(
	ctx context.Context, user store.User,
	sessionID uuid.UUID,
	examineeID uuid.UUID,
	data MarkResponseBody,
) error {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := svc.q.WithTx(tx)

	// -------------------------------------------------------------
	// --- Ensuring User (as Examiner) owns the Script -------------

	if session, err := qtx.FindSessionByID(ctx, sessionID); err != nil {
		return err
	} else if err := common.RequireOwner(user, session.CreatorID); err != nil {
		return err
	}

	// -------------------------------------------------------------
	// --- Range through the given marks ---------------------------

	for _, mark := range data.Marks {
		if err := qtx.ExaminerMarkQuestionResponse(ctx, store.ExaminerMarkQuestionResponseParams{
			QuestionID: mark.QuestionID,
			SessionID:  sessionID,
			ExamineeID: examineeID,
			Mark:       mark.Value,
			Feedback:   mark.Feedback,
		}); err != nil {
			log.Printf("Error in ExaminerMarkSubmission: Examiner (%v)", err)
			return apperr.ErrInternal
		}
	}

	// -------------------------------------------------------------
	// --- Fails if the examiner didn't mark all questions ---------

	if _, err := qtx.SetResponseStatusPostManualMark(
		ctx,
		store.SetResponseStatusPostManualMarkParams{
			SessionID:  sessionID,
			ExamineeID: examineeID,
		},
	); err != nil {
		log.Printf("Error in ExaminerMarkSubmission: SetStatus (%v)", err)
		// TODO: Change to indicate that the examiner didn't mark all questions
		return apperr.ErrBadRequest
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error in ExaminerMarkSubmission: (%v)", err)
		return apperr.ErrInternal
	}

	return nil
}
