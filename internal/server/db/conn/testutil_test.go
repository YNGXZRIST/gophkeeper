package conn

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMockConnDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return &DB{DB: db}, mock
}

func requireTxDone(t *testing.T, tx *sql.Tx) {
	t.Helper()
	_ = tx.Rollback()
}
