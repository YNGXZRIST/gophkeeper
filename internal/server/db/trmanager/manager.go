package trmanager

import (
	"context"
	"database/sql"
	"gophermart-loyalty/internal/gopherman/db/conn"
	"gophermart-loyalty/internal/gopherman/db/retryable"
)

// Manager coordinates transactional execution.
type Manager struct {
	*conn.DB
}

// Tx wraps sql transaction.
type Tx struct {
	*sql.Tx
}

// NewManager creates transaction manager for DB connection.
func NewManager(db *conn.DB) *Manager {
	return &Manager{db}
}

// WithinTx runs callback within a retried transaction.
func (m *Manager) WithinTx(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	_, err := retryable.RunWithRetry(ctx, func() (struct{}, error) {
		sqlTx, err := m.DB.BeginTx(ctx, opts)
		if err != nil {
			return struct{}{}, err
		}
		tx := &Tx{Tx: sqlTx}
		defer func() { _ = tx.Rollback() }()
		txCtx := WithTx(ctx, tx)
		if err := fn(txCtx); err != nil {
			return struct{}{}, err
		}
		if err := tx.Commit(); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

// WithoutTx runs callback without opening a transaction.
func (m *Manager) WithoutTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	return nil
}
