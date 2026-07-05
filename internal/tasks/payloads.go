package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	TypeStartSessionEvent = "session:start"
	TypeEndSessionEvent   = "session:end"
)

// Start Session Payload

type StartSessionPayload struct {
	SessionID string
}

func NewStartSessionTask(id string) (*asynq.Task, error) {
	payload, err := json.Marshal(StartSessionPayload{SessionID: id})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeStartSessionEvent, payload), nil
}

// End Session Payload

type EndSessionPayload struct {
	SessionID string
}

func NewEndSessionTask(id string) (*asynq.Task, error) {
	payload, err := json.Marshal(EndSessionPayload{SessionID: id})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeEndSessionEvent, payload), nil
}
