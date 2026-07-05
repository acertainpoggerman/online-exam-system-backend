package worker

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"

// 	"github.com/acertainpoggerman/online-exam-system/internal/tasks"
// 	"github.com/hibiken/asynq"
// )

// type Handler struct {
// 	service *services.Service
// }

// func NewHandler(svc *services.Service) *Handler {
// 	return &Handler{service: svc}
// }

// // Task Handler Functions

// func (h *Handler) HandleStartSessionTask(ctx context.Context, t *asynq.Task) error {

// 	var p tasks.StartSessionPayload
// 	if err := json.Unmarshal(t.Payload(), &p); err != nil {
// 		return fmt.Errorf("json.Unmarshal failed, %v: %w", err, asynq.SkipRetry)
// 	}

// 	return h.service.StartSessionByID(ctx, p.SessionID)
// }

// func (h *Handler) HandleEndSessionTask(ctx context.Context, t *asynq.Task) error {

// 	var p tasks.EndSessionPayload
// 	if err := json.Unmarshal(t.Payload(), &p); err != nil {
// 		return fmt.Errorf("json.Unmarshal failed, %v: %w", err, asynq.SkipRetry)
// 	}

// 	return h.service.EndSessionByID(ctx, p.SessionID)
// }
