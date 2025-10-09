package models

import "time"

// [User-Based Structs]

type RegisterRequestBody struct {
	Role     string `json:"role"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// [Script-Based Structs]

type CreateScriptRequestBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateScriptRequestBody struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Questions   []Question `json:"questions"`
}

// [Session-Based Structs]

type CreateSessionRequestBody struct {
	ScriptID          string              `json:"script_id"`
	DurationMins      uint                `json:"duration_mins"`
	GraceDurationMins uint                `json:"grace_duration_mins"`
	StartTime         Optional[time.Time] `json:"start_time"`
}

type UpdateSessionRequestBody struct {
	Status            SessionStatus       `json:"status"`
	StartTime         Optional[time.Time] `json:"start_time"`
	DurationMins      uint                `json:"duration_mins"`
	GraceDurationMins uint                `json:"grace_duration_mins"`
}
