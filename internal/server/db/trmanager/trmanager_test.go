package trmanager

import (
	"context"
	"errors"
	"gophermart-loyalty/internal/gopherman/db/conn"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithTx_TxFromContext_roundTrip(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	t.Cleanup(func() { _ = mock.ExpectationsWereMet() })

	mock.ExpectBegin()
	sqlTx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tx := &Tx{Tx: sqlTx}

	child := WithTx(ctx, tx)
	got, ok := TxFromContext(child)
	if !ok || got != tx {
		t.Fatalf("TxFromContext = (%v, %v), want (same tx, true)", got, ok)
	}

	if _, ok := TxFromContext(ctx); ok {
		t.Fatal("parent context must not carry tx")
	}

	_ = sqlTx.Rollback()
}

func TestResolve_returnsDBOrTx(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	t.Cleanup(func() { _ = mock.ExpectationsWereMet() })

	cdb := &conn.DB{DB: db}
	ctx := t.Context()

	if got := Resolve(ctx, cdb); got != cdb {
		t.Fatalf("Resolve without tx = %T, want *conn.DB", got)
	}

	mock.ExpectBegin()
	sqlTx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	tx := &Tx{Tx: sqlTx}
	txCtx := WithTx(ctx, tx)

	if got := Resolve(txCtx, cdb); got != tx {
		t.Fatalf("Resolve with tx = %T, want *conn.Tx", got)
	}

	_ = sqlTx.Rollback()
}

func TestManager_WithinTx_success_commits(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	cdb := &conn.DB{DB: db}
	m := NewManager(cdb)

	var sawTx bool
	err = m.WithinTx(t.Context(), nil, func(ctx context.Context) error {
		_, sawTx = TxFromContext(ctx)
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}
	if !sawTx {
		t.Fatal("callback context must carry *conn.Tx")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManager_WithinTx_fnError_rollbacks(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	cdb := &conn.DB{DB: db}
	m := NewManager(cdb)

	fnErr := errors.New("fn failed")
	err = m.WithinTx(t.Context(), nil, func(ctx context.Context) error {
		return fnErr
	})
	if !errors.Is(err, fnErr) {
		t.Fatalf("WithinTx err = %v, want %v", err, fnErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManager_WithinTx_beginError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	beginErr := errors.New("begin failed")
	mock.ExpectBegin().WillReturnError(beginErr)

	cdb := &conn.DB{DB: db}
	m := NewManager(cdb)

	err = m.WithinTx(t.Context(), nil, func(ctx context.Context) error {
		t.Fatal("fn must not run when BeginTx fails")
		return nil
	})
	if !errors.Is(err, beginErr) {
		t.Fatalf("WithinTx err = %v, want %v", err, beginErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManager_WithinTx_commitError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	commitErr := errors.New("commit failed")
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(commitErr)

	cdb := &conn.DB{DB: db}
	m := NewManager(cdb)

	err = m.WithinTx(t.Context(), nil, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, commitErr) {
		t.Fatalf("WithinTx err = %v, want %v", err, commitErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestManager_WithoutTx(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	cdb := &conn.DB{DB: db}
	m := NewManager(cdb)
	ctx := t.Context()

	t.Run("nil_error", func(t *testing.T) {
		calls := 0
		err := m.WithoutTx(ctx, func(c context.Context) error {
			calls++
			if c != ctx {
				t.Fatal("context must be passed through")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithoutTx: %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("returns_error", func(t *testing.T) {
		want := errors.New("x")
		err := m.WithoutTx(ctx, func(context.Context) error {
			return want
		})
		if !errors.Is(err, want) {
			t.Fatalf("err = %v, want %v", err, want)
		}
	})
}

func TestQuerier_implementedByConnTypes(t *testing.T) {
	t.Parallel()

	var _ Querier = (*conn.DB)(nil)
	var _ Querier = (*Tx)(nil)
}
