package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/client/auth"
)

type SessionRepo struct {
	repoBase
}

var ErrNoSession = errors.New("no session")

type ErrParseToken struct {
	Reason error
}

func newErrParseToken(reason error) *ErrParseToken {
	return &ErrParseToken{Reason: reason}
}

func (e ErrParseToken) Error() string {
	return fmt.Sprintf("parse token: %v", e.Reason)
}

func NewSessionRepo(conn *sql.DB) *SessionRepo {
	return &SessionRepo{repoBase: repoBase{db: conn}}
}

type SessionRaw struct {
	login        string
	accessToken  string
	refreshToken string
}

func (sr *SessionRepo) Get(ctx context.Context) (*auth.Session, error) {
	raw := &SessionRaw{}
	err := sr.db.QueryRowContext(ctx, "SELECT login,access_token,refresh_token FROM session").Scan(raw.login, raw.accessToken, raw.refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	accessToken, err := auth.NewToken(raw.accessToken)
	if err != nil {
		return nil, newErrParseToken(fmt.Errorf("access token: %w", err))
	}
	refreshToken, err := auth.NewToken(raw.refreshToken)
	if err != nil {
		return nil, newErrParseToken(fmt.Errorf("refresh token: %w", err))
	}
	return auth.NewSession(raw.login, accessToken, refreshToken), nil
}
