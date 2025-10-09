package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Script struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatorID bson.ObjectID `json:"creator_id" bson:"creator_id"`

	Title       string     `json:"title" bson:"title"`
	Description string     `json:"description" bson:"description"`
	Questions   []Question `json:"questions" bson:"questions"`

	OpCode         string              `json:"-" bson:"operator_code"`
	CreatedAt      time.Time           `json:"-" bson:"created_at"`
	DeletedAt      Optional[time.Time] `json:"-" bson:"deleted_at"`
	LastModifiedAt Optional[time.Time] `json:"-" bson:"modified_at"`
}

// [Question-Based Types]

type QuestionType string

const (
	QuestionMultipleChoice QuestionType = "question.multiple_choice"
	QuestionCheckBox       QuestionType = "question.checkbox"
	QuestionShortAnswer    QuestionType = "question.short_answer"
	QuestionParagraph      QuestionType = "question.paragraph"
)

// Base Question Type

type BaseQuestion struct {
	Type   QuestionType `json:"type" bson:"type"`
	Prompt string       `json:"prompt" bson:"prompt"`
	Mark   uint         `json:"mark" bson:"mark"`
}

func (q *BaseQuestion) GetType() QuestionType { return q.Type }

// Question Types (Structs)

type MultipleChoiceQuestion struct {
	BaseQuestion `bson:",inline"`
	Options      []string `json:"options" bson:"options"`
	Answer       uint     `json:"answer" bson:"answer"`
}

type CheckBoxQuestion struct {
	BaseQuestion `bson:",inline"`
	Options      []string `json:"options" bson:"options"`
	Answers      []uint   `json:"answers" bson:"answers"`
}

type ShortAnswerQuestion struct {
	BaseQuestion `bson:",inline"`
	Answer       string  `json:"answer" bson:"answer"`
	AccThreshold float32 `json:"acc_threshold" bson:"acc_threshold"`
}

// type ParagraphQuestion struct {
// 	BaseQuestion `bson:",inline"`
// }
