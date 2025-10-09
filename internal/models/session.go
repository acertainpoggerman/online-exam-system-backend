package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// SessionStatus represents the current state of a Session. Important in
// determining if a session can be accessed, and who can access information
// about the session.
//
// Can be any of the following states:
//
//   - StatusSessionInactive: Used to represent the state of a session that has not been started or scheduled to start
//   - StatusSessionPending: Used to represent the state of a session that has been scheduled to start
//   - StatusSessionOngoing: Used to represent the state of a session that has been started
//   - StatusSessionPaused: Used to represent the state of an ongoing session that has been halted
//   - StatusSessionCompleted: Used to represent the state of a session that has ended
type SessionStatus string

const (
	StatusSessionInactive  SessionStatus = "session.inactive"
	StatusSessionPending   SessionStatus = "session.pending"
	StatusSessionOngoing   SessionStatus = "session.ongoing"
	StatusSessionPaused    SessionStatus = "session.paused"
	StatusSessionCompleted SessionStatus = "session.completed"
)

type Session struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	ScriptID  bson.ObjectID `json:"script_id" bson:"script_id"`
	CreatorID bson.ObjectID `json:"creator_id" bson:"creator_id"`

	Status            SessionStatus       `json:"status" bson:"status"`
	StartTime         Optional[time.Time] `json:"start_time" bson:"start_time"`
	DurationMins      uint                `json:"duration_mins" bson:"duration_mins"`
	GraceDurationMins uint                `json:"grace_duration_mins" bson:"grace_duration_mins"`

	StartSessionTaskID Optional[string] `json:"-" bson:"start_task_id"`
	EndSessionTaskID   Optional[string] `json:"-" bson:"end_task_id"`

	OpCode         string              `json:"-" bson:"operator_code"`
	CreatedAt      time.Time           `json:"-" bson:"created_at"`
	DeletedAt      Optional[time.Time] `json:"-" bson:"deleted_at"`
	LastModifiedAt Optional[time.Time] `json:"-" bson:"modified_at"`
}
