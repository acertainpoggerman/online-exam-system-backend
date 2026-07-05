package svcerrors

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *FieldError) Error() string { return e.Message }
