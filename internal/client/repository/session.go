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

const GetSessionSQL = `SELECT login,access_token,refresh_token FROM session`
const SetSessionSQL = `INSERT INTO session (id, login, access_token, refresh_token, updated_at)
VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    login = excluded.login,
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    updated_at = CURRENT_TIMESTAMP`

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
	err := sr.db.QueryRowContext(ctx, GetSessionSQL).Scan(&raw.login, &raw.accessToken, &raw.refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	return sr.initSession(raw.login, raw.accessToken, raw.refreshToken)

}
func (sr *SessionRepo) Save(ctx context.Context, login, accessToken, refreshToken string) (*auth.Session, error) {
	_, err := sr.db.ExecContext(ctx, SetSessionSQL, login, accessToken, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}
	return sr.initSession(login, accessToken, refreshToken)
}

func (sr *SessionRepo) initSession(l, a, r string) (*auth.Session, error) {
	accessToken, err := auth.NewToken(a)
	if err != nil {
		return nil, newErrParseToken(fmt.Errorf("access token: %w", err))
	}
	refreshToken, err := auth.NewToken(r)
	if err != nil {
		return nil, newErrParseToken(fmt.Errorf("refresh token: %w", err))
	}
	return auth.NewSession(l, accessToken, refreshToken), nil

}
