package model

import "time"

// PasswordData is the decrypted plaintext payload of a password secret,
// serialized and encrypted before it leaves the client.
type PasswordData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Meta     string `json:"meta"`
}

// Password is the full client-side password: server-held metadata plus the decrypted payload.
type Password struct {
	ID        string
	Data      PasswordData
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
