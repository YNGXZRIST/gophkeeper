package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	Raw       string
	ExpiresAt time.Time
}

func NewToken(raw string) (Token, error) {
	t := Token{Raw: raw}
	claims := jwt.RegisteredClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(raw, claims)
	if err != nil {
		return t, fmt.Errorf("failed to parse token: %w", err)
	}
	t.ExpiresAt = claims.ExpiresAt.Time
	return t, nil
}
func (t Token) Expired(skew time.Duration) bool {
	return time.Now().Add(skew).Before(t.ExpiresAt)
}
