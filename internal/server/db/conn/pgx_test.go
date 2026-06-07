package conn

import (
	"errors"
	"gophermart-loyalty/internal/gopherman/config/db"
	"gophermart-loyalty/internal/gopherman/db/retryable"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDB_Exec(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		D, m := newMockConnDB(t)
		query := "DELETE FROM users WHERE id=$1"
		m.ExpectExec(query).
			WithArgs(int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		res, err := D.Exec(query, int64(1))
		if err != nil {
			t.Fatalf("Exec() error = %v, want nil", err)
		}
		if res == nil {
			t.Fatalf("Exec() res = nil, want non-nil")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock expectations not met: %v", err)
		}
	})

	t.Run("non_retriable_error_wrapped", func(t *testing.T) {
		D, m := newMockConnDB(t)
		query := "INSERT INTO users(login) VALUES ($1)"
		m.ExpectExec(query).
			WithArgs("test").
			WillReturnError(errors.New("boom"))

		_, err := D.Exec(query, "test")
		if err == nil {
			t.Fatalf("Exec() error = nil, want non-nil")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock expectations not met: %v", err)
		}
	})
}

func TestDB_ExecContext(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		D, m := newMockConnDB(t)
		query := "UPDATE users SET balance=$1 WHERE id=$2"
		m.ExpectExec(query).
			WithArgs(10.5, int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		res, err := D.ExecContext(ctx, query, 10.5, int64(2))
		if err != nil {
			t.Fatalf("ExecContext() error = %v, want nil", err)
		}
		if res == nil {
			t.Fatalf("ExecContext() res = nil, want non-nil")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock expectations not met: %v", err)
		}
	})

	t.Run("non_retriable_error_wrapped", func(t *testing.T) {
		D, m := newMockConnDB(t)
		query := "UPDATE users SET login=$1 WHERE id=$2"
		m.ExpectExec(query).
			WithArgs("bob", int64(1)).
			WillReturnError(errors.New("boom"))

		_, err := D.ExecContext(ctx, query, "bob", int64(1))
		if err == nil {
			t.Fatalf("ExecContext() error = nil, want non-nil")
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock expectations not met: %v", err)
		}
	})
}

func TestDB_Query(t *testing.T) {
	D, m := newMockConnDB(t)
	query := "SELECT id FROM users WHERE id=$1"
	m.ExpectQuery(query).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rows, err := D.Query(query, int64(1))
	if err != nil {
		t.Fatalf("Query() error = %v, want nil", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("Query() no rows, want 1 row")
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		t.Fatalf("rows.Scan() error = %v", err)
	}
	if id != 1 {
		t.Fatalf("Query() id = %d, want 1", id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}
}

func TestDB_QueryContext(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		D, m := newMockConnDB(t)
		query := "SELECT login FROM users WHERE id=$1"
		m.ExpectQuery(query).
			WithArgs(int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"login"}).AddRow("test"))

		rows, err := D.QueryContext(ctx, query, int64(10))
		if err != nil {
			t.Fatalf("QueryContext() error = %v, want nil", err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Fatalf("QueryContext() no rows, want 1 row")
		}
		var login string
		if err := rows.Scan(&login); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		if login != "test" {
			t.Fatalf("QueryContext() login = %q, want %q", login, "test")
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err() error = %v", err)
		}
		if err := m.ExpectationsWereMet(); err != nil {
			t.Fatalf("sqlmock expectations not met: %v", err)
		}
	})
}

func TestNewConn(t *testing.T) {
	cfg := db.NewCfg("test")
	conn, err := NewConn(cfg)
	if err != nil {
		t.Fatalf("NewConn() error = %v, want nil", err)
	}
	if conn == nil {
		t.Fatalf("NewConn() conn = nil, want non-nil")
	}
	if conn.DNS != "test" {
		t.Fatalf("NewConn() DNS = %q, want %q", conn.DNS, "test")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("conn.Close() error = %v, want nil", err)
	}
}

func Test_retryable_RunWithRetry(t *testing.T) {
	ctx := t.Context()

	t.Run("non_retriable_error_wrapped", func(t *testing.T) {
		_, err := retryable.RunWithRetry[int](ctx, func() (int, error) {
			return 0, errors.New("boom")
		})
		if err == nil {
			t.Fatalf("RunWithRetry() error = nil, want non-nil")
		}
	})

	t.Run("success_return_value", func(t *testing.T) {
		got, err := retryable.RunWithRetry[int](ctx, func() (int, error) {
			return 7, nil
		})
		if err != nil {
			t.Fatalf("RunWithRetry() error = %v, want nil", err)
		}
		if got != 7 {
			t.Fatalf("RunWithRetry() got = %d, want %d", got, 7)
		}
	})
}
