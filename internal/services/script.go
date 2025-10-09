package services

import (
	"context"
	"time"

	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (svc *Service) CreateScript(
	ctx context.Context,
	creatorID string,
	data models.CreateScriptRequestBody,
) (*models.Script, error) {

	cID, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, err
	}

	script, err := svc.repo.CreateScript(ctx, models.Script{
		// [Script Data]
		CreatorID:   cID,
		Title:       data.Title,
		Description: data.Description,
		// [PTA]
		OpCode:    "system",
		CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return script, err
}

// Getting script
func (svc *Service) GetScriptByID(ctx context.Context, scriptID string) (*models.Script, error) {

	script, err := svc.repo.FindScriptByID(ctx, scriptID)
	if err != nil {
		return nil, err
	}

	return script, nil
}

// Deleting script
func (svc *Service) DeleteScriptByID(ctx context.Context, scriptID string) error {

	err := svc.repo.DeleteScriptByID(ctx, scriptID)
	return err
}

// Creating Question for Script
func (svc *Service) UpdateScriptByID(
	ctx context.Context,
	scriptID string,
	data models.UpdateScriptRequestBody,
) (*models.Script, error) {

	sID, err := bson.ObjectIDFromHex(scriptID)
	if err != nil {
		return nil, err
	}

	script, err := svc.repo.UpdateScript(ctx, models.Script{
		ID:          sID,
		Title:       data.Title,
		Description: data.Description,
		Questions:   data.Questions,
	})
	if err != nil {
		return nil, err
	}

	return script, nil
}
