package model

import "time"

// RefreshToken is a stored refresh token: only its hash is persisted, never the
// plaintext value handed to the client.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
