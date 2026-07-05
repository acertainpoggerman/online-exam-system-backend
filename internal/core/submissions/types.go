package submissions

import (
	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/core/scripts"
	"github.com/acertainpoggerman/online-exam-system/internal/update"
	"github.com/google/uuid"
)

type CreateSubmissionBody struct {
	SessionID uuid.UUID `json:"session_id"`
}

type Submission struct {
	store.Submission
	Examinee store.User `json:"examinee,omitzero"`
	Answers  []Answer   `json:"answers,omitempty"`
}

type Answer struct {
	store.Answer
	Question scripts.Question `json:"question,omitzero"`
	Value    []string         `json:"value"`
}

type PatchSubmissionRequest struct {
	IncludeSubmission bool                  `json:"include_submission_in_response"`
	Actions           []update.UpdateAction `json:"actions"`
}
