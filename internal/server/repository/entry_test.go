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

const entryTestTable = "entries"

var entryColumns = []string{"id", "user_id", "data", "version", "created_at", "updated_at"}

func newEntryTestRepo(db *sql.DB) *EntryRepo {
	return NewEntryRepo(&conn.DB{DB: db}, entryTestTable)
}

func TestEntryRepoGetByUser(t *testing.T) {
	query := buildListByUserQuery(entryTestTable)
	tests := []struct {
		name      string
		lastID    string
		mockFn    func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "success with cursor",
			lastID: "e0",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("a"), int64(1), time.Now(), time.Now()).
					AddRow("e2", "u1", []byte("b"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("u1", "e0", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 2,
		},
		{
			name: "empty cursor",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns)
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("u1", nil, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 0,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("u1", nil, 10, 0).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("a"), "bad", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
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

			repo := newEntryTestRepo(db)

			var entries []*model.Entry
			for entry, e := range repo.GetByUser(context.Background(), "u1", tt.lastID, 10, 0) {
				if e != nil {
					err = e
					break
				}
				entries = append(entries, entry)
			}

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, entries, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEntryRepoGetByID(t *testing.T) {
	query := buildGetByIDQuery(entryTestTable)
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
					WillReturnRows(rows)
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrEntryNotFound,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
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

			repo := newEntryTestRepo(db)
			entry, err := repo.GetByID(context.Background(), "u1", "e1")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, entry)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "e1", entry.ID)
				assert.Equal(t, []byte("data"), entry.Data)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEntryRepoCreate(t *testing.T) {
	query := buildCreateQuery(entryTestTable)
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1", []byte("data")).
					WillReturnRows(rows)
			},
		},
		{
			name: "upsert on conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("data"), int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1", []byte("data")).
					WillReturnRows(rows)
			},
		},
		{
			name: "no rows",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1", []byte("data")).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1", []byte("data")).
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

			repo := newEntryTestRepo(db)
			entry, err := repo.Create(context.Background(), "u1", &model.Entry{ID: "e1", Data: []byte("data")})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, entry)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "e1", entry.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEntryRepoUpdate(t *testing.T) {
	query := buildUpdateQuery(entryTestTable)
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(entryColumns).
					AddRow("e1", "u1", []byte("new"), int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs([]byte("new"), "e1", "u1", int64(1)).
					WillReturnRows(rows)
			},
		},
		{
			name: "version conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs([]byte("new"), "e1", "u1", int64(1)).
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrVersionConflict,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs([]byte("new"), "e1", "u1", int64(1)).
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

			repo := newEntryTestRepo(db)
			entry, err := repo.Update(context.Background(), "u1", &model.Entry{ID: "e1", Data: []byte("new"), Version: 1})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, entry)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(2), entry.Version)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEntryRepoDelete(t *testing.T) {
	query := buildDeleteQuery(entryTestTable)
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			sentinel: model.ErrEntryNotFound,
			wantErr:  true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(query)).
					WithArgs("e1", "u1").
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

			repo := newEntryTestRepo(db)
			err = repo.Delete(context.Background(), "u1", "e1")

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

func TestEntryRepoChanges(t *testing.T) {
	query := buildChangesQuery(entryTestTable)
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
					AddRow("e1", []byte("a"), int64(1), false, time.Now()).
					AddRow("e2", []byte("b"), int64(2), true, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
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
					AddRow("e1", []byte("a"), int64(1), false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("u1", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs("u1", nil).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("e1", []byte("a"), "bad", false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(query)).
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

			repo := newEntryTestRepo(db)

			var changes []*model.EntryChange
			for ch, e := range repo.Changes(context.Background(), "u1", tt.since) {
				if e != nil {
					err = e
					break
				}
				changes = append(changes, ch)
			}

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
