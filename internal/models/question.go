package models

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Question struct {
	Content IQuestion
}

type IQuestion interface {
	GetType() QuestionType
}

func (q *Question) UnmarshalBSON(data []byte) error {

	var raw bson.M
	if err := bson.Unmarshal(data, &raw); err != nil {
		return err
	}

	qTypeStr, ok := (raw["type"].(string))
	if !ok {
		return fmt.Errorf("missing \"type\" field in question")
	}

	switch qType := QuestionType(qTypeStr); qType {

	case QuestionMultipleChoice:
		{
			var question *MultipleChoiceQuestion
			if err := bson.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	case QuestionCheckBox:
		{
			var question *CheckBoxQuestion
			if err := bson.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	case QuestionShortAnswer:
		{
			var question *ShortAnswerQuestion
			if err := bson.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	default:
		return fmt.Errorf("unknown question type: %s", qType)
	}

	return nil
}

func (q *Question) UnmarshalJSON(data []byte) error {

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	qTypeStr, ok := (raw["type"].(string))
	if !ok {
		return fmt.Errorf("missing \"type\" field in question")
	}

	switch qType := QuestionType(qTypeStr); qType {

	case QuestionMultipleChoice:
		{
			var question *MultipleChoiceQuestion
			if err := json.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	case QuestionCheckBox:
		{
			var question *CheckBoxQuestion
			if err := json.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	case QuestionShortAnswer:
		{
			var question *ShortAnswerQuestion
			if err := json.Unmarshal(data, &question); err != nil {
				return err
			}
			q.Content = question
		}

	default:
		return fmt.Errorf("unknown question type: %s", qType)
	}

	return nil
}

func (q *Question) MarshalBSON() ([]byte, error) {
	return bson.Marshal(q.Content)
}

func (q *Question) MarshalJSON() ([]byte, error) {
	return json.Marshal(q.Content)
}
