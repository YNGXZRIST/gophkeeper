package repository

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/client/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(exp)}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return signed
}

func TestSessionGetNoSession(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)

	_, err := repo.Get(context.Background())
	require.ErrorIs(t, err, ErrNoSession)
}

func TestSessionSaveAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)
	ctx := context.Background()

	access := makeJWT(t, time.Now().Add(time.Hour))
	cred := auth.Credentials{
		Login:        "user@example.com",
		AccessToken:  access,
		RefreshToken: "refresh-raw",
		EncSalt:      []byte("salt"),
		WrappedDek:   []byte("dek"),
	}

	saved, err := repo.Save(ctx, cred)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", saved.Login)
	assert.Equal(t, access, saved.Access.Raw)
	assert.Equal(t, "refresh-raw", saved.Refresh.Raw)
	assert.Equal(t, []byte("salt"), saved.EncSalt)
	assert.Equal(t, []byte("dek"), saved.WrappedDek)

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", got.Login)
	assert.Equal(t, access, got.Access.Raw)
	assert.Equal(t, []byte("dek"), got.WrappedDek)
}

func TestSessionSaveUpsert(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)
	ctx := context.Background()

	first := makeJWT(t, time.Now().Add(time.Hour))
	second := makeJWT(t, time.Now().Add(2*time.Hour))

	_, err := repo.Save(ctx, auth.Credentials{Login: "a", AccessToken: first, RefreshToken: "r1"})
	require.NoError(t, err)
	_, err = repo.Save(ctx, auth.Credentials{Login: "b", AccessToken: second, RefreshToken: "r2"})
	require.NoError(t, err)

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "b", got.Login)
	assert.Equal(t, second, got.Access.Raw)
	assert.Equal(t, "r2", got.Refresh.Raw)
}

func TestSessionSaveInvalidToken(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)
	ctx := context.Background()

	_, err := repo.Save(ctx, auth.Credentials{Login: "a", AccessToken: "not-a-jwt", RefreshToken: "r"})
	require.Error(t, err)
	var parseErr *ErrParseToken
	assert.ErrorAs(t, err, &parseErr)
}

func TestSessionGetInvalidStoredToken(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, SetSessionSQL, "a", "garbage", "r", []byte("s"), []byte("d"))
	require.NoError(t, err)

	_, err = repo.Get(ctx)
	require.Error(t, err)
	var parseErr *ErrParseToken
	assert.ErrorAs(t, err, &parseErr)
}

func TestSessionClear(t *testing.T) {
	db := newTestDB(t)
	repo := NewSessionRepo(db)
	ctx := context.Background()

	access := makeJWT(t, time.Now().Add(time.Hour))
	_, err := repo.Save(ctx, auth.Credentials{Login: "a", AccessToken: access, RefreshToken: "r"})
	require.NoError(t, err)

	require.NoError(t, repo.Clear(ctx))
	_, err = repo.Get(ctx)
	require.ErrorIs(t, err, ErrNoSession)
}

func TestErrParseTokenMessage(t *testing.T) {
	e := newErrParseToken(assert.AnError)
	assert.Contains(t, e.Error(), "parse token")
}
