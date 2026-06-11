package model

import "time"

type User struct {
	ID        string
	Login     string
	Pass      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
