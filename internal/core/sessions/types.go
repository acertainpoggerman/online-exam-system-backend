package sessions

import (
	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/google/uuid"
)

type CreateSessionBody struct {
	Title         string    `json:"title"`
	ScriptID      uuid.UUID `json:"script_id"`
	QuestionCount *int32    `json:"question_count"`
}

type Session struct {
	store.Session
	// Schedule store.SessionSchedule `json:"schedule"
}

type EventType string

const (
	EventTypeSessionStarted EventType = "session:started"
	EventTypeSessionEnded   EventType = "session:ended"
)

type SessionEvent struct {
	Type EventType `json:"event_type"`
	Data any       `json:"data,omitempty"`
}

func generateSessionCode(n int) (code string) {
	chars := "0123456789abcdefghijklmnopqrstuvwxyz"
	for i := range n {
		code += string(chars[i])
	}
	return
}
