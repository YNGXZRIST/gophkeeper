package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenRepoCreate(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "created_at"}).
					AddRow("t1", "u1", "hash", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(RefreshTokenCreateQuery)).
					WithArgs("u1", "hash", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(RefreshTokenCreateQuery)).
					WithArgs("u1", "hash", sqlmock.AnyArg()).
					WillReturnError(errors.New("boom"))
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

			repo := NewRefreshTokenRepo(&conn.DB{DB: db})
			rt, err := repo.Create(context.Background(), model.RefreshToken{
				UserID: "u1", TokenHash: "hash", ExpiresAt: time.Now(),
			})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, rt)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "t1", rt.ID)
				assert.Equal(t, "u1", rt.UserID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRefreshTokenRepoDeleteByUserID(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(RefreshTokenDeleteByUserIDQuery)).
					WithArgs("u1").
					WillReturnResult(sqlmock.NewResult(0, 2))
			},
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(RefreshTokenDeleteByUserIDQuery)).
					WithArgs("u1").
					WillReturnError(errors.New("boom"))
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

			repo := NewRefreshTokenRepo(&conn.DB{DB: db})
			err = repo.DeleteByUserID(context.Background(), "u1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
