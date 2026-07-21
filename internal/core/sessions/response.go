package sessions

import (
	"context"
	"errors"
	"log"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/acertainpoggerman/online-exam-system/internal/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (svc *sessionService) FindExamineeLogs(
	ctx context.Context, user store.User,
	sessionID uuid.UUID,
	examineeID uuid.UUID,
) ([]store.ProctorEvent, error) {

	if err := common.RequireRole(user, store.UserRoleExaminer); err != nil {
		return []store.ProctorEvent{}, err
	}

	events, err := svc.q.FindProctorLogsForExaminee(ctx, store.FindProctorLogsForExamineeParams{
		SessionID:  sessionID,
		ExamineeID: examineeID,
	})
	if err != nil {
		return []store.ProctorEvent{}, err
	}

	return events, nil
}

// ----------------------------------------------------------
// --- Response-based Methods -------------------------------
// ----------------------------------------------------------

func (svc *sessionService) ExamineeFindResponses(ctx context.Context, user store.User) ([]Response, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return []Response{}, err
	}

	responses, err := svc.q.ExamineeFindResponses(ctx, user.ID)
	if err != nil {
		return []Response{}, err
	}

	var submissions []Response

	for _, resp := range responses {

		if resp.Status != store.ResponseStatusMarked {
			submissions = append(submissions, Response{Response: resp})
			continue
		}

		marks, err := svc.q.FindResponseMarks(ctx, store.FindResponseMarksParams{
			SessionID:  resp.SessionID,
			ExamineeID: resp.ExamineeID,
		})
		if err != nil {
			return []Response{}, err
		}

		submissions = append(submissions, Response{
			Response:    resp,
			TotalMark:   marks.Total,
			MaximumMark: marks.Maximum,
		})
	}

	return submissions, nil
}

func (svc *sessionService) ExamineeFindSessionResponse(ctx context.Context, user store.User, sessionID uuid.UUID) (Response, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return Response{}, apperr.ErrInternal
	}

	resp, err := svc.q.FindSingleResponse(ctx, store.FindSingleResponseParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		return Response{}, apperr.ErrInternal
	}

	response := Response{Response: resp}

	rows, err := svc.q.FindQuestionResponses(ctx, store.FindQuestionResponsesParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		return Response{}, apperr.ErrInternal
	}

	for _, row := range rows {

		getAnswerKey := response.Status == store.ResponseStatusMarked

		question, err := svc.script.FindQuestionByID(
			ctx,
			row.QuestionResponse.QuestionID,
			getAnswerKey,
			true,
		)
		if err != nil {
			log.Printf("Error: Question %v\n", err)
			return Response{}, apperr.ErrInternal
		}

		response.QuestionResponses = append(
			response.QuestionResponses,
			QuestionResponse{
				QuestionResponse: row.QuestionResponse,
				Value:            row.Value,
				Question:         question,
			},
		)
	}

	return response, nil
}

func (svc *sessionService) UpdateQuestionResponse(ctx context.Context, user store.User, sessionID uuid.UUID, questionID uuid.UUID, qr QuestionResponse) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	if err := svc.q.ReplaceQuestionResponseValues(ctx, store.ReplaceQuestionResponseValuesParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
		QuestionID: questionID,
		Value:      qr.Value,
	}); err != nil {
		log.Printf("Error in UpdateSubmissionAnswer: (%v)", err)
		return apperr.ErrInternal
	}

	return nil
}

func (svc *sessionService) SubmitSessionResponse(ctx context.Context, user store.User, sessionID uuid.UUID) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	if _, err := svc.q.SetResponseSubmitted(ctx, store.SetResponseSubmittedParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrNotFound
		}
		log.Printf("Error SubmitSubmission: %v", err)
		return apperr.ErrInternal
	}

	svc.hub.BroadcastTo(sessionID, Message{
		Type: MessageTypeParticipantsChanged,
	}, store.UserRoleExaminer)

	svc.hub.RemoveMember(user.ID, sessionID)

	return nil
}
