package model

import "time"

type Note struct {
	ID        string
	UserID    string
	Data      []byte
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NoteChange struct {
	ID        string
	Data      []byte
	Version   int64
	Deleted   bool
	UpdatedAt time.Time
}
