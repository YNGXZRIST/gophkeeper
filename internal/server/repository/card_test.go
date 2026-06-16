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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cardColumns = []string{"id", "user_id", "data", "version", "created_at", "updated_at"}

func TestCardRepoGetByID(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cardColumns).
					AddRow("c1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardGetByIDQuery)).
					WithArgs("c1", "u1").
					WillReturnRows(rows)
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardGetByIDQuery)).
					WithArgs("c1", "u1").
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrCardNotFound,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardGetByIDQuery)).
					WithArgs("c1", "u1").
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

			repo := NewCardRepo(&conn.DB{DB: db})
			card, err := repo.GetByID(context.Background(), &model.User{ID: "u1"}, "c1")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, card)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "c1", card.ID)
				assert.Equal(t, []byte("data"), card.Data)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCardRepoUpdate(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cardColumns).
					AddRow("c1", "u1", []byte("new"), int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardUpdateQuery)).
					WithArgs([]byte("new"), "c1", "u1", int64(1)).
					WillReturnRows(rows)
			},
		},
		{
			name: "version conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardUpdateQuery)).
					WithArgs([]byte("new"), "c1", "u1", int64(1)).
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrVersionConflict,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardUpdateQuery)).
					WithArgs([]byte("new"), "c1", "u1", int64(1)).
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

			repo := NewCardRepo(&conn.DB{DB: db})
			card, err := repo.Update(context.Background(), &model.User{ID: "u1"},
				&model.Card{ID: "c1", Data: []byte("new"), Version: 1})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, card)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(2), card.Version)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCardRepoDelete(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(CardDeleteQuery)).
					WithArgs("c1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(CardDeleteQuery)).
					WithArgs("c1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			sentinel: model.ErrCardNotFound,
			wantErr:  true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(CardDeleteQuery)).
					WithArgs("c1", "u1").
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

			repo := NewCardRepo(&conn.DB{DB: db})
			err = repo.Delete(context.Background(), &model.User{ID: "u1"}, "c1")

			if tt.wantErr {
				require.Error(t, err)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
