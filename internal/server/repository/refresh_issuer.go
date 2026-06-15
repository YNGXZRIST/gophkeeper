package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/server/auth/refresh"
	"gophkeeper/internal/server/model"
	"time"
)

// DefaultRefreshTTL is how long an issued refresh token stays valid.
const DefaultRefreshTTL = 30 * 24 * time.Hour

// RefreshIssuer generates refresh tokens, stores their hash and returns the
// plaintext. It honours an ambient transaction from ctx, so it works both
// inside a transaction (registration) and standalone (login).
type RefreshIssuer struct {
	tokens *RefreshTokenRepo
	secret []byte
	ttl    time.Duration
}

// NewRefreshIssuer constructs a refresh-token issuer.
func NewRefreshIssuer(tokens *RefreshTokenRepo, secret []byte, ttl time.Duration) *RefreshIssuer {
	if ttl <= 0 {
		ttl = DefaultRefreshTTL
	}
	return &RefreshIssuer{tokens: tokens, secret: secret, ttl: ttl}
}

// Issue creates a new refresh token for the user, persists its hash and returns
// the plaintext token to hand to the client.
func (i *RefreshIssuer) Issue(ctx context.Context, userID string) (string, error) {
	plain, err := refresh.Generate()
	if err != nil {
		return "", fmt.Errorf("generate refresh: %w", err)
	}
	if _, err := i.tokens.Create(ctx, model.RefreshToken{
		UserID:    userID,
		TokenHash: refresh.Hash(plain, i.secret),
		ExpiresAt: time.Now().Add(i.ttl),
	}); err != nil {
		return "", err
	}
	return plain, nil
}

// Rotate validates a plaintext refresh token, revokes it and issues a fresh one
// for the same user, returning the user ID and the new plaintext token. An
// unknown or expired token yields model.ErrInvalidRefreshToken.
func (i *RefreshIssuer) Rotate(ctx context.Context, plain string) (string, string, error) {
	hash := refresh.Hash(plain, i.secret)
	rt, err := i.tokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", model.ErrInvalidRefreshToken
		}
		return "", "", err
	}
	if time.Now().After(rt.ExpiresAt) {
		return "", "", model.ErrInvalidRefreshToken
	}
	if err := i.tokens.DeleteByHash(ctx, hash); err != nil {
		return "", "", err
	}
	newPlain, err := i.Issue(ctx, rt.UserID)
	if err != nil {
		return "", "", err
	}
	return rt.UserID, newPlain, nil
}
