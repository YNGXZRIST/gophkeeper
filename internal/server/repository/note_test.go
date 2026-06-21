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

var noteColumns = []string{"id", "user_id", "data", "version", "created_at", "updated_at"}

func TestNoteRepoGetByUser(t *testing.T) {
	tests := []struct {
		name      string
		lastID    string
		mockFn    func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "success with cursor",
			lastID: "n0",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("a"), int64(1), time.Now(), time.Now()).
					AddRow("n2", "u1", []byte("b"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteListByUserQuery)).
					WithArgs("u1", "n0", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 2,
		},
		{
			name: "empty cursor",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns)
				m.ExpectQuery(regexp.QuoteMeta(NoteListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 0,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("a"), "bad", time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteListByUserQuery)).
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

			repo := NewNoteRepo(&conn.DB{DB: db})
			notes, err := repo.GetByUser(context.Background(), &model.User{ID: "u1"}, tt.lastID, 10, 0)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, notes, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNoteRepoGetByID(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteGetByIDQuery)).
					WithArgs("n1", "u1").
					WillReturnRows(rows)
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteGetByIDQuery)).
					WithArgs("n1", "u1").
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrNoteNotFound,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteGetByIDQuery)).
					WithArgs("n1", "u1").
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

			repo := NewNoteRepo(&conn.DB{DB: db})
			note, err := repo.GetByID(context.Background(), &model.User{ID: "u1"}, "n1")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, note)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "n1", note.ID)
				assert.Equal(t, []byte("data"), note.Data)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNoteRepoCreate(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("data"), int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteCreateQuery)).
					WithArgs("n1", "u1", []byte("data")).
					WillReturnRows(rows)
			},
		},
		{
			name: "upsert on conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("data"), int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteCreateQuery)).
					WithArgs("n1", "u1", []byte("data")).
					WillReturnRows(rows)
			},
		},
		{
			name: "no rows",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteCreateQuery)).
					WithArgs("n1", "u1", []byte("data")).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteCreateQuery)).
					WithArgs("n1", "u1", []byte("data")).
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

			repo := NewNoteRepo(&conn.DB{DB: db})
			note, err := repo.Create(context.Background(), &model.User{ID: "u1"}, &model.Note{ID: "n1", Data: []byte("data")})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, note)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "n1", note.ID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNoteRepoUpdate(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(noteColumns).
					AddRow("n1", "u1", []byte("new"), int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteUpdateQuery)).
					WithArgs([]byte("new"), "n1", "u1", int64(1)).
					WillReturnRows(rows)
			},
		},
		{
			name: "version conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteUpdateQuery)).
					WithArgs([]byte("new"), "n1", "u1", int64(1)).
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrVersionConflict,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteUpdateQuery)).
					WithArgs([]byte("new"), "n1", "u1", int64(1)).
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

			repo := NewNoteRepo(&conn.DB{DB: db})
			note, err := repo.Update(context.Background(), &model.User{ID: "u1"}, &model.Note{ID: "n1", Data: []byte("new"), Version: 1})

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, note)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(2), note.Version)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNoteRepoDelete(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(NoteDeleteQuery)).
					WithArgs("n1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(NoteDeleteQuery)).
					WithArgs("n1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			sentinel: model.ErrNoteNotFound,
			wantErr:  true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(NoteDeleteQuery)).
					WithArgs("n1", "u1").
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

			repo := NewNoteRepo(&conn.DB{DB: db})
			err = repo.Delete(context.Background(), &model.User{ID: "u1"}, "n1")

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

func TestNoteRepoChanges(t *testing.T) {
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
					AddRow("n1", []byte("a"), int64(1), false, time.Now()).
					AddRow("n2", []byte("b"), int64(2), true, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteChangesQuery)).
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
					AddRow("n1", []byte("a"), int64(1), false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteChangesQuery)).
					WithArgs("u1", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(NoteChangesQuery)).
					WithArgs("u1", nil).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("n1", []byte("a"), "bad", false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(NoteChangesQuery)).
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

			repo := NewNoteRepo(&conn.DB{DB: db})
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
