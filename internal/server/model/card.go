package model

import "time"

type Card struct {
	ID        string
	UserID    string
	Data      []byte
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CardChange struct {
	ID        string
	Data      []byte
	Version   int64
	Deleted   bool
	UpdatedAt time.Time
}
