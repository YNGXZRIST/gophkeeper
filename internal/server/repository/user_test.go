package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepoCreate(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "login", "password", "created_at", "updated_at"}).
					AddRow("u1", "alice", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(UserRegisterQuery)).
					WithArgs("alice", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
		},
		{
			name: "duplicate login",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(UserRegisterQuery)).
					WithArgs("alice", sqlmock.AnyArg()).
					WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation})
			},
			sentinel: model.ErrLoginTaken,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(UserRegisterQuery)).
					WithArgs("alice", sqlmock.AnyArg()).
					WillReturnError(errors.New("connection refused"))
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

			repo := NewUserRepo(&conn.DB{DB: db})
			user, err := repo.Create(context.Background(), model.User{Login: "alice", Pass: "secret"})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "u1", user.ID)
				assert.Equal(t, "alice", user.Login)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepoGetByLogin(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "found",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "login", "password", "created_at", "updated_at"}).
					AddRow("u1", "alice", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(UserGetByLoginQuery)).
					WithArgs("alice").
					WillReturnRows(rows)
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(UserGetByLoginQuery)).
					WithArgs("alice").
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: sql.ErrNoRows,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.mockFn(mock)

			repo := NewUserRepo(&conn.DB{DB: db})
			user, err := repo.GetByLogin(context.Background(), "alice")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "u1", user.ID)
				assert.Equal(t, "hash", user.Pass)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
