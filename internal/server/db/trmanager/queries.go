// Package trmanager provides helpers for transactional query execution.
package trmanager

import (
	"context"
	"database/sql"
	"gophermart-loyalty/internal/gopherman/db/conn"
)

// Querier describes minimal DB methods used by repositories.
type Querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Resolve returns transaction from context or falls back to base DB.
func Resolve(ctx context.Context, db *conn.DB) Querier {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return db
}
