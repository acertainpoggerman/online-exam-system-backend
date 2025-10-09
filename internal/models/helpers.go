package models

import "time"

// Intended to be used with structs
type Optional[T any] *T

// Generalized PTA Struct that can be embedded
type PointInTimeArch struct {
	OpCode         string              `json:"-" bson:"operator_code"`
	CreatedAt      time.Time           `json:"-" bson:"created_at"`
	DeletedAt      Optional[time.Time] `json:"-" bson:"deleted_at"`
	LastModifiedAt Optional[time.Time] `json:"-" bson:"modified_at"`
}
