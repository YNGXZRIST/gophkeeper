package model

import "time"

type Entry struct {
	ID        string
	UserID    string
	Data      []byte
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
type EntryChange struct {
	ID        string
	Data      []byte
	Version   int64
	Deleted   bool
	UpdatedAt time.Time
}
