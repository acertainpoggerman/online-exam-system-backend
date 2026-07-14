package apperr

type FieldError struct {
	Fields  []string `json:"fields"`
	Message string   `json:"message"`
	Code    string   `json:"code"`
}

func (e *FieldError) Error() string { return e.Message }
