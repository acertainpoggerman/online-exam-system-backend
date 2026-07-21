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
// --- Submission-based Methods -----------------------------
// ----------------------------------------------------------

func (svc *sessionService) ExamineeFindSubmissions(ctx context.Context, user store.User) ([]Submission, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return []Submission{}, err
	}

	subs, err := svc.q.ExamineeFindSubmitted(ctx, user.ID)
	if err != nil {
		return []Submission{}, err
	}

	var submissions []Submission

	for _, sub := range subs {

		if sub.Status != store.SubmissionStatusMarked {
			submissions = append(submissions, Submission{Submission: sub})
			continue
		}

		marks, err := svc.q.SubmissionMarks(ctx, store.SubmissionMarksParams{
			SessionID:  sub.SessionID,
			ExamineeID: sub.ExamineeID,
		})
		if err != nil {
			return []Submission{}, err
		}

		submissions = append(submissions, Submission{
			Submission: sub,
			TotalMark:  int(marks.Lhs),
			MaxMark:    int(marks.Rhs),
		})
	}

	return submissions, nil
}

func (svc *sessionService) ExamineeFindSubmission(ctx context.Context, user store.User, sessionID uuid.UUID) (Submission, error) {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return Submission{}, apperr.ErrInternal
	}

	sub, err := svc.q.FindSubmissionByID(ctx, store.FindSubmissionByIDParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		return Submission{}, apperr.ErrInternal
	}

	submission := Submission{Submission: sub}

	answerFields, err := svc.q.FindSubmissionAnswers(ctx, store.FindSubmissionAnswersParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		return Submission{}, apperr.ErrInternal
	}

	for idx, answer := range answerFields {

		getAnswerKey := submission.Status == store.SubmissionStatusMarked

		question, err := svc.script.FindQuestionByID(ctx, answer.SubmissionQuestion.QuestionID, getAnswerKey)
		if err != nil {
			log.Printf("Error: Question %v\n", err)
			return Submission{}, apperr.ErrInternal
		}

		submission.Answers = append(submission.Answers, Answer{
			SubmissionQuestion: answer.SubmissionQuestion,
			Value:              answer.Value,
			Question:           question,
		})

		if submission.Answers[idx].Question.Mark == nil {
			mark, err := svc.q.GetQuestionMark(ctx, question.ID)
			if err != nil {
				log.Printf("Error: QuestionMark %v\n", err)
				return Submission{}, apperr.ErrInternal
			}
			submission.Answers[idx].Question.Mark = &mark
		}
	}

	return submission, nil
}

func (svc *sessionService) UpdateAnswer(ctx context.Context, user store.User, sessionID uuid.UUID, questionID uuid.UUID, answer Answer) error {

	if err := common.RequireRole(user, store.UserRoleExaminee); err != nil {
		return err
	}

	sub, err := svc.q.FindSubmissionByID(ctx, store.FindSubmissionByIDParams{
		SessionID:  sessionID,
		ExamineeID: user.ID,
	})
	if err != nil {
		log.Printf("Error in UpdateSubmissionAnswer: (%v)", err)
		return apperr.ErrInternal
	} else if err := common.RequireOwner(user, sub.ExamineeID); err != nil {
		return err
	}

	if err := svc.q.ReplaceAnswerForQuestion(ctx, store.ReplaceAnswerForQuestionParams{
		SubmissionID: sub.ID,
		QuestionID:   questionID,
		Value:        answer.Value,
	}); err != nil {
		log.Printf("Error in UpdateSubmissionAnswer: (%v)", err)
		return apperr.ErrInternal
	}

	return nil
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

	svc.hub.BroadcastTo(sessionID, Message{
		Type: MessageTypeParticipantsChanged,
	}, store.UserRoleExaminer)

	svc.hub.RemoveMember(user.ID, sessionID)

	return nil
}
