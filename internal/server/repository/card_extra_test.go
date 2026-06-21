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

func TestCardRepoGetByUser(t *testing.T) {
	tests := []struct {
		name      string
		lastID    string
		mockFn    func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "success with cursor",
			lastID: "c0",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cardColumns).
					AddRow("c1", "u1", []byte("a"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardListByUserQuery)).
					WithArgs("u1", "c0", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "empty cursor",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnRows(sqlmock.NewRows(cardColumns))
			},
			wantCount: 0,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cardColumns).
					AddRow("c1", "u1", []byte("a"), "bad", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnRows(rows)
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
			cards, err := repo.GetByUser(context.Background(), &model.User{ID: "u1"}, tt.lastID, 10, 0)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, cards, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCardRepoCreate(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cardColumns).
					AddRow("c1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardCreateQuery)).
					WithArgs(sqlmock.AnyArg(), "u1", []byte("data")).
					WillReturnRows(rows)
			},
		},
		{
			name: "no rows",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardCreateQuery)).
					WithArgs(sqlmock.AnyArg(), "u1", []byte("data")).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardCreateQuery)).
					WithArgs(sqlmock.AnyArg(), "u1", []byte("data")).
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
			card, err := repo.Create(context.Background(), &model.User{ID: "u1"}, &model.Card{ID: "c1", Data: []byte("data")})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, card)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "c1", card.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCardRepoChanges(t *testing.T) {
	changeColumns := []string{"id", "data", "version", "deleted", "updated_at"}
	tests := []struct {
		name      string
		since     time.Time
		mockFn    func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:  "since zero",
			since: time.Time{},
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("c1", []byte("a"), int64(1), false, time.Now()).
					AddRow("c2", []byte("b"), int64(2), true, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardChangesQuery)).
					WithArgs("u1", nil).
					WillReturnRows(rows)
			},
			wantCount: 2,
		},
		{
			name:  "since non-zero",
			since: time.Now(),
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("c1", []byte("a"), int64(1), false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardChangesQuery)).
					WithArgs("u1", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(CardChangesQuery)).
					WithArgs("u1", nil).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("c1", []byte("a"), "bad", false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(CardChangesQuery)).
					WithArgs("u1", nil).
					WillReturnRows(rows)
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
			changes, err := repo.Changes(context.Background(), &model.User{ID: "u1"}, tt.since)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, changes, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
