package model

import "time"

type User struct {
	ID        int64
	Login     string
	Pass      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
