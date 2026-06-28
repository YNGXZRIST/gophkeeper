package model

import "time"

type User struct {
	ID          string
	Login       string
	Pass        string
	EncSalt     []byte
	WrappedDesk []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
