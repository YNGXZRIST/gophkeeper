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

const GetSessionSQL = `SELECT login,access_token,refresh_token,enc_salt,wrapped_dek FROM session`
const ClearSessionSQL = `DELETE FROM session`
const SetSessionSQL = `INSERT INTO session (id, login, access_token, refresh_token, enc_salt, wrapped_dek, updated_at)
VALUES (1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    login = excluded.login,
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    enc_salt = excluded.enc_salt,
    wrapped_dek = excluded.wrapped_dek,
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
	encSalt      []byte
	wrappedDek   []byte
}

func (sr *SessionRepo) Get(ctx context.Context) (*auth.Session, error) {
	raw := &SessionRaw{}
	err := sr.db.QueryRowContext(ctx, GetSessionSQL).Scan(&raw.login, &raw.accessToken, &raw.refreshToken, &raw.encSalt, &raw.wrappedDek)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	return sr.initSession(raw.login, raw.accessToken, raw.refreshToken, raw.encSalt, raw.wrappedDek)

}
func (sr *SessionRepo) Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error) {
	_, err := sr.db.ExecContext(ctx, SetSessionSQL, cred.Login, cred.AccessToken, cred.RefreshToken, cred.EncSalt, cred.WrappedDek)
	if err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}
	return sr.initSession(cred.Login, cred.AccessToken, cred.RefreshToken, cred.EncSalt, cred.WrappedDek)
}

func (sr *SessionRepo) Clear(ctx context.Context) error {
	if _, err := sr.db.ExecContext(ctx, ClearSessionSQL); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}

func (sr *SessionRepo) initSession(l, a, r string, encSalt, wrappedDek []byte) (*auth.Session, error) {
	accessToken, err := auth.NewToken(a)
	if err != nil {
		return nil, newErrParseToken(fmt.Errorf("access token: %w", err))
	}
	refreshToken := auth.Token{Raw: r}
	return auth.NewSession(l, accessToken, refreshToken, encSalt, wrappedDek), nil

}
