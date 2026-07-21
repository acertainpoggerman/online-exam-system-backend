package sessions

import (
	"math/rand/v2"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/core/examscripts"
	"github.com/google/uuid"
)

type CreateSessionBody struct {
	Title    string    `json:"title"`
	ScriptID uuid.UUID `json:"script_id"`
}

type JoinSessionBody struct {
	JoinCode string `json:"code"`
}

type MarkResponseBody struct {
	Marks []ResponseMark `json:"response_marks"`
}

type ResponseMark struct {
	QuestionID uuid.UUID `json:"question_id"`
	Value      int32     `json:"value"`
	Feedback   string    `json:"feedback"`
}

type Session struct {
	store.Session
	// Schedule store.SessionSchedule `json:"schedule"
}

// ---------------------------------------------------------
// --- Submissions -----------------------------------------
// ---------------------------------------------------------

type CreateSubmissionBody struct {
	SessionID uuid.UUID `json:"session_id"`
}

type Response struct {
	store.Response
	Examinee          store.User         `json:"examinee,omitzero"`
	TotalMark         int64              `json:"total_mark"`
	MaximumMark       int64              `json:"maximum_mark"`
	QuestionResponses []QuestionResponse `json:"question_responses,omitempty"`
}

type QuestionResponse struct {
	store.QuestionResponse
	Question examscripts.Question `json:"question,omitzero"`
	Value    []string             `json:"value"`
}

// ---------------------------------------------------------
// --- Helper Functions ------------------------------------
// ---------------------------------------------------------

func generateSessionCode(n int) (code string) {
	chars := "0123456789abcdefghijklmnopqrstuvwxyz"
	for _ = range n {
		i := rand.IntN(len(chars))
		code += string(chars[i])
	}
	return
}
