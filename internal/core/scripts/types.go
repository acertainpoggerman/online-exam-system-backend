package scripts

import (
	"encoding/json"
	"fmt"

	store "github.com/acertainpoggerman/online-exam-system/internal/adapters/postgresql/sqlc"
	"github.com/acertainpoggerman/online-exam-system/internal/update"
	"github.com/go-viper/mapstructure/v2"
)

type CreateScriptBody struct {
	Title       string `json:"title"`
	Heading     string `json:"heading"`
	Description string `json:"description"`
}

type UpdateScriptBody struct {
	CreateScriptBody
	DefaultMark int `json:"default_mark"`
}

type Script struct {
	store.Script
	Questions []Question `json:"questions"`
}

// --------------------------------------------------------------------------
// --- QUESTION -------------------------------------------------------------
// --------------------------------------------------------------------------

// Using composition, you can create an interface that contains a private
// function only implementable by structures within the same package, creating
// a way to have a "union" of sub-types. For purposes of unmarshaling/marshaling,
// a "type" field is kept to be able to determine what type of sub-type to
// unmarshal to.

type Question struct {
	store.Question `mapstructure:",squash"`
	AnswerKey      []string    `json:"answer_key,omitzero"`
	SubQuestion    SubQuestion `json:"sub_question"`
}
type SubQuestion interface{ subQuestion() }

// --- QuestionType: Choice Question ----------------------------------------

type ChoiceQuestion struct {
	store.ChoiceQuestion `mapstructure:",squash"`
	Options              []store.Option `json:"options"`
}

func (q *ChoiceQuestion) subQuestion() {}

// --- QuestionType: Text Question ------------------------------------------

type TextQuestion struct {
	store.TextQuestion `mapstructure:",squash"`
}

func (q *TextQuestion) subQuestion() {}

// --------------------------------------------------------------------------
// --- QUESTIONS : Handling Conversions -------------------------------------
// --------------------------------------------------------------------------

// Thanks to Composition, Marshal functions are not necessary as whatever
// is converting the Go type to the target data does not need to know what
// type the sub-type (Question.SubQuestion) is.
//
// Unmarshaling is as simple as unmarshaling the shared information
// (store.Question), keeping the sub-type information in the raw state
// and converting that after determining the intended sub-type to be
// unmarshaled into.
//
// Getting what type the sub-type is for performing operations based on type
// can be done by switching Question.SubQuestion.(type) and using the concrete
// types as switch cases.

// --- json.Unmarshal -------------------------------------------------------

func (q *Question) UnmarshalJSON(data []byte) error {

	var raw struct {
		store.Question
		AnswerKey   []string        `json:"answer_key"`
		SubQuestion json.RawMessage `json:"sub_question"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	q.Question = raw.Question
	q.AnswerKey = raw.AnswerKey

	switch q.Type {
	case store.QuestionTypeChoice:
		q.SubQuestion = &ChoiceQuestion{}
	case store.QuestionTypeText:
		q.SubQuestion = &TextQuestion{}
	default:
		return fmt.Errorf("Question Type: \"%s\" does not exist", q.Type)
	}

	return json.Unmarshal(raw.SubQuestion, q.SubQuestion)
}

// --- mapstructure.Decode --------------------------------------------------

func (q *Question) UnmarshalMapstructure(data any) error {

	var raw struct {
		store.Question `mapstructure:",squash"`
		AnswerKey      []string       `mapstructure:"answer_key"`
		SubQuestion    map[string]any `mapstructure:"sub_question"`
	}
	if err := mapstructure.Decode(data, &raw); err != nil {
		return err
	}
	q.Question = raw.Question

	switch q.Type {
	case store.QuestionTypeChoice:
		q.SubQuestion = &ChoiceQuestion{}
	case store.QuestionTypeText:
		q.SubQuestion = &TextQuestion{}
	default:
		return fmt.Errorf("Question Type: \"%s\" does not exist", q.Type)
	}

	return mapstructure.Decode(raw.SubQuestion, q.SubQuestion)
}
