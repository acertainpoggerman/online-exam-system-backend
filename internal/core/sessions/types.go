package sessions

import (
	"math/rand/v2"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/google/uuid"
)

type CreateSessionBody struct {
	Title         string    `json:"title"`
	ScriptID      uuid.UUID `json:"script_id"`
	QuestionCount *int32    `json:"question_count"`
}

type JoinSessionBody struct {
	JoinCode string `json:"code"`
}

type Session struct {
	store.Session
	// Schedule store.SessionSchedule `json:"schedule"
}

func generateSessionCode(n int) (code string) {
	chars := "0123456789abcdefghijklmnopqrstuvwxyz"
	for _ = range n {
		i := rand.IntN(len(chars))
		code += string(chars[i])
	}
	return
}
