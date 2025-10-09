package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID           bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Role         string        `json:"role" bson:"role"`
	Email        string        `json:"email" bson:"email"`
	PasswordHash string        `json:"-" bson:"password_hash"`

	// FirstName string `json:"fname" bson:"fname"`
	// LastName  string `json:"lname" bson:"lname"`

	OpCode         string              `json:"-" bson:"operator_code"`
	CreatedAt      time.Time           `json:"-" bson:"created_at"`
	DeletedAt      Optional[time.Time] `json:"-" bson:"deleted_at"`
	LastModifiedAt Optional[time.Time] `json:"-" bson:"modified_at"`
}

type Permission string

type Role struct {
	ID          string   `json:"-" bson:"_id"`
	Name        string   `json:"name" bson:"name"`
	Permissions []string `json:"permissions" bson:"permissions"`
}
