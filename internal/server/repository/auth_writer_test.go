package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/server/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAuthWriter(db *conn.DB) *AuthWriter {
	repos := Repositories{
		User:         NewUserRepo(db),
		RefreshToken: NewRefreshTokenRepo(db),
	}
	issuer := NewRefreshIssuer(repos.RefreshToken, []byte("secret"), time.Hour)
	return NewAuthWriter(trmanager.NewManager(db), repos, issuer)
}

func TestAuthWriterRegister(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success commits",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				userRows := sqlmock.NewRows([]string{"id", "login", "password", "created_at", "updated_at"}).
					AddRow("u1", "alice", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(UserRegisterQuery)).
					WithArgs("alice", sqlmock.AnyArg()).
					WillReturnRows(userRows)
				tokenRows := sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "created_at"}).
					AddRow("t1", "u1", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(RefreshTokenCreateQuery)).
					WithArgs("u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(tokenRows)
				m.ExpectCommit()
			},
		},
		{
			name: "user insert fails rolls back",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(regexp.QuoteMeta(UserRegisterQuery)).
					WithArgs("alice", sqlmock.AnyArg()).
					WillReturnError(errors.New("boom"))
				m.ExpectRollback()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.mockFn(mock)

			aw := newAuthWriter(&conn.DB{DB: db})
			user, plain, err := aw.Register(context.Background(), model.User{Login: "alice", Pass: "secret"})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.Empty(t, plain)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "u1", user.ID)
				assert.NotEmpty(t, plain)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAuthWriterIssueRefresh(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "created_at"}).
		AddRow("t1", "u1", "hash", time.Now(), time.Now())
	mock.ExpectQuery(regexp.QuoteMeta(RefreshTokenCreateQuery)).
		WithArgs("u1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	aw := newAuthWriter(&conn.DB{DB: db})
	plain, err := aw.IssueRefresh(context.Background(), "u1")

	require.NoError(t, err)
	assert.NotEmpty(t, plain)
	assert.NoError(t, mock.ExpectationsWereMet())
}
