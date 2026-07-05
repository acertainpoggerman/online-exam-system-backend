package api

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Cursor struct {
	ID uuid.UUID
	Ts time.Time
}

func EncodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (*Cursor, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c Cursor
	return &c, json.Unmarshal(b, &c)
}
