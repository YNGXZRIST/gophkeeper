package model

import "time"

type Password struct {
	ID        string
	UserID    string
	Data      []byte
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
