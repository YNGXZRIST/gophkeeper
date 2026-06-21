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

var fileColumns = []string{"id", "user_id", "meta", "chunk_count", "version", "created_at", "updated_at"}

func TestFileRepoGetByUser(t *testing.T) {
	tests := []struct {
		name      string
		lastID    string
		mockFn    func(sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
	}{
		{
			name:   "success with cursor",
			lastID: "f0",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(fileColumns).
					AddRow("f1", "u1", []byte("m"), 3, int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileListByUserQuery)).
					WithArgs("u1", "f0", 10, 0).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "empty cursor",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnRows(sqlmock.NewRows(fileColumns))
			},
			wantCount: 0,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileListByUserQuery)).
					WithArgs("u1", nil, 10, 0).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(fileColumns).
					AddRow("f1", "u1", []byte("m"), "bad", int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileListByUserQuery)).
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

			repo := NewFileRepo(&conn.DB{DB: db})
			files, err := repo.GetByUser(context.Background(), &model.User{ID: "u1"}, tt.lastID, 10, 0)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, files, tt.wantCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoGetMeta(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(fileColumns).
					AddRow("f1", "u1", []byte("meta"), 2, int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileGetMetaQuery)).
					WithArgs("f1", "u1").
					WillReturnRows(rows)
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileGetMetaQuery)).
					WithArgs("f1", "u1").
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrFileNotFound,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileGetMetaQuery)).
					WithArgs("f1", "u1").
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

			repo := NewFileRepo(&conn.DB{DB: db})
			file, err := repo.GetMeta(context.Background(), &model.User{ID: "u1"}, "f1")

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, file)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, "f1", file.ID)
				assert.Equal(t, 2, file.ChunkCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoCreateFile(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(fileColumns).
					AddRow("f1", "u1", []byte("meta"), 3, int64(1), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileCreateQuery)).
					WithArgs("u1", []byte("meta"), 3).
					WillReturnRows(rows)
			},
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileCreateQuery)).
					WithArgs("u1", []byte("meta"), 3).
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

			repo := NewFileRepo(&conn.DB{DB: db})
			file, err := repo.CreateFile(context.Background(), &model.User{ID: "u1"}, []byte("meta"), 3)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "f1", file.ID)
				assert.Equal(t, 3, file.ChunkCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoInsertChunk(t *testing.T) {
	tests := []struct {
		name    string
		mockFn  func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(FileChunkInsertQuery)).
					WithArgs("f1", 0, []byte("chunk")).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(FileChunkInsertQuery)).
					WithArgs("f1", 0, []byte("chunk")).
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

			repo := NewFileRepo(&conn.DB{DB: db})
			err = repo.InsertChunk(context.Background(), "f1", 0, []byte("chunk"))

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoStreamChunks(t *testing.T) {
	tests := []struct {
		name      string
		mockFn    func(sqlmock.Sqlmock)
		fn        func(idx int, data []byte) error
		wantCount int
		wantErr   bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"idx", "data"}).
					AddRow(0, []byte("a")).
					AddRow(1, []byte("b"))
				m.ExpectQuery(regexp.QuoteMeta(FileChunksByFileQuery)).
					WithArgs("f1").
					WillReturnRows(rows)
			},
			wantCount: 2,
		},
		{
			name: "callback error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"idx", "data"}).
					AddRow(0, []byte("a"))
				m.ExpectQuery(regexp.QuoteMeta(FileChunksByFileQuery)).
					WithArgs("f1").
					WillReturnRows(rows)
			},
			fn:      func(idx int, data []byte) error { return errors.New("callback failed") },
			wantErr: true,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileChunksByFileQuery)).
					WithArgs("f1").
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"idx", "data"}).
					AddRow("bad", []byte("a"))
				m.ExpectQuery(regexp.QuoteMeta(FileChunksByFileQuery)).
					WithArgs("f1").
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

			repo := NewFileRepo(&conn.DB{DB: db})
			var count int
			fn := tt.fn
			if fn == nil {
				fn = func(idx int, data []byte) error {
					count++
					return nil
				}
			}
			err = repo.StreamChunks(context.Background(), "f1", fn)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoUpdateMeta(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(fileColumns).
					AddRow("f1", "u1", []byte("new"), 3, int64(2), time.Now(), time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileUpdateMetaQuery)).
					WithArgs([]byte("new"), "f1", "u1", int64(1)).
					WillReturnRows(rows)
			},
		},
		{
			name: "version conflict",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileUpdateMetaQuery)).
					WithArgs([]byte("new"), "f1", "u1", int64(1)).
					WillReturnError(sql.ErrNoRows)
			},
			sentinel: model.ErrVersionConflict,
			wantErr:  true,
		},
		{
			name: "other db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileUpdateMetaQuery)).
					WithArgs([]byte("new"), "f1", "u1", int64(1)).
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

			repo := NewFileRepo(&conn.DB{DB: db})
			file, err := repo.UpdateMeta(context.Background(), &model.User{ID: "u1"}, "f1", []byte("new"), 1)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, file)
				if tt.sentinel != nil {
					assert.ErrorIs(t, err, tt.sentinel)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, int64(2), file.Version)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFileRepoDelete(t *testing.T) {
	tests := []struct {
		name     string
		mockFn   func(sqlmock.Sqlmock)
		sentinel error
		wantErr  bool
	}{
		{
			name: "success",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(FileDeleteQuery)).
					WithArgs("f1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "not found",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(FileDeleteQuery)).
					WithArgs("f1", "u1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			sentinel: model.ErrFileNotFound,
			wantErr:  true,
		},
		{
			name: "db error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectExec(regexp.QuoteMeta(FileDeleteQuery)).
					WithArgs("f1", "u1").
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

			repo := NewFileRepo(&conn.DB{DB: db})
			err = repo.Delete(context.Background(), &model.User{ID: "u1"}, "f1")

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

func TestFileRepoChanges(t *testing.T) {
	changeColumns := []string{"id", "meta", "version", "deleted", "updated_at"}
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
					AddRow("f1", []byte("a"), int64(1), false, time.Now()).
					AddRow("f2", []byte("b"), int64(2), true, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileChangesQuery)).
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
					AddRow("f1", []byte("a"), int64(1), false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileChangesQuery)).
					WithArgs("u1", sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantCount: 1,
		},
		{
			name: "query error",
			mockFn: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(FileChangesQuery)).
					WithArgs("u1", nil).
					WillReturnError(errors.New("connection refused"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			mockFn: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(changeColumns).
					AddRow("f1", []byte("a"), "bad", false, time.Now())
				m.ExpectQuery(regexp.QuoteMeta(FileChangesQuery)).
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

			repo := NewFileRepo(&conn.DB{DB: db})
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
