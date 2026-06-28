// Package token issues and validates signed JWT access tokens.
package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultAccessTTL is how long an issued access token stays valid.
const DefaultAccessTTL = 15 * time.Minute

var signingMethod = jwt.SigningMethodHS256

// Issuer signs and verifies access tokens with a shared HMAC secret.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer constructs an access-token issuer.
func NewIssuer(secret []byte, ttl time.Duration) *Issuer {
	if ttl <= 0 {
		ttl = DefaultAccessTTL
	}
	return &Issuer{secret: secret, ttl: ttl}
}

// Issue returns a signed access token whose subject is the user ID.
func (i *Issuer) Issue(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	}
	signed, err := jwt.NewWithClaims(signingMethod, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// Parse verifies the token signature and expiry and returns its subject (user ID).
func (i *Issuer) Parse(tokenString string) (string, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			if t.Method != signingMethod {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return i.secret, nil
		},
	)
	if err != nil {
		return "", fmt.Errorf("parse access token: %w", err)
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return "", fmt.Errorf("invalid access token")
	}
	return claims.Subject, nil
}
