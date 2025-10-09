package services

import (
	"context"
	"time"

	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (svc *Service) CreateSession(
	ctx context.Context,
	creatorID string,
	data models.CreateSessionRequestBody,
) (*models.Session, error) {

	sID, err := bson.ObjectIDFromHex(data.ScriptID)
	if err != nil {
		return nil, err
	}

	cID, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}

	session, err := svc.repo.CreateSession(ctx, models.Session{
		// [Session Data]
		CreatorID:         cID,
		ScriptID:          sID,
		Status:            models.StatusSessionInactive,
		DurationMins:      data.DurationMins,
		GraceDurationMins: data.GraceDurationMins,
		// [PTA]
		CreatedAt: time.Now(),
		OpCode:    "system",
	})
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (svc *Service) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {

	session, err := svc.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (svc *Service) DeleteSessionByID(ctx context.Context, sessionID string) error {

	err := svc.repo.DeleteSessionByID(ctx, sessionID)
	return err
}

func (svc *Service) UpdateSessionByID(
	ctx context.Context,
	sessionID string,
	data models.UpdateSessionRequestBody,
) (*models.Session, error) {

	sID, err := bson.ObjectIDFromHex(sessionID)
	if err != nil {
		return nil, err
	}

	session, err := svc.repo.UpdateSession(ctx, models.Session{
		ID:                sID,
		Status:            data.Status,
		DurationMins:      data.DurationMins,
		GraceDurationMins: data.GraceDurationMins,
		StartTime:         data.StartTime,
	})
	if err != nil {
		return nil, err
	}

	return session, nil
}
