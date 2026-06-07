// Package conn provides SQL connection wrapper with retry helpers.
package conn

import (
	"context"
	"database/sql"
	"fmt"
	"gophkeeper/internal/server/db/retryable"
	"gophkeeper/internal/shared/errors/labelerrors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	// DBLabel is base label for DB-related errors.
	DBLabel = "DB"
	// PGXLabel is label prefix for pgx connection operations.
	PGXLabel = DBLabel + ".PGX"
)

// DB wraps sql.DB and DB configuration.
type DB struct {
	*sql.DB
	*Config
}

// NewConn opens new PostgreSQL connection.
func NewConn(cfg *Config) (*DB, error) {
	if cfg == nil || cfg.DSN == "" {
		return nil, labelerrors.NewLabelError(PGXLabel+".NewConn.DSN", fmt.Errorf("database DSN is not set"))
	}
	dsn := cfg.DSN
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, labelerrors.NewLabelError(PGXLabel+".NewConn.Open", fmt.Errorf("error connecting to database: %w", err))
	}
	return &DB{DB: conn, Config: cfg}, nil
}

// ExecContext executes SQL statement with retries.
func (D *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return retryable.RunWithRetry(ctx, func() (sql.Result, error) {
		return D.DB.ExecContext(ctx, query, args...)
	})
}

// Exec executes SQL statement with background context and retries.
func (D *DB) Exec(query string, args ...any) (sql.Result, error) {
	return retryable.RunWithRetry(context.Background(), func() (sql.Result, error) {
		return D.DB.Exec(query, args...)
	})
}

// Query runs SQL query with background context and retries.
func (D *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return retryable.RunWithRetry(context.Background(), func() (*sql.Rows, error) {
		return D.DB.Query(query, args...)
	})
}

// QueryContext runs SQL query with retries.
func (D *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return retryable.RunWithRetry(ctx, func() (*sql.Rows, error) {
		return D.DB.QueryContext(ctx, query, args...)
	})
}
