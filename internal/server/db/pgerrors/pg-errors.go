package pgerrors

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// PgErrors is a typed wrapper around postgres error details.
type PgErrors struct {
	Code    string // SQLSTATE, например "23505", "42P01"
	Message string
	Err     error
}

// Error returns formatted postgres error details.
func (e PgErrors) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("code=%s %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("code=%s %s", e.Code, e.Message)
}

// Unwrap returns wrapped original error.
func (e PgErrors) Unwrap() error { return e.Err }

// NewPgError converts incoming error to PgErrors wrapper.
func NewPgError(err error) error {
	if err == nil {
		return nil
	}
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return &PgErrors{Code: pgErr.Code, Message: pgErr.Message, Err: err}
	}
	return &PgErrors{Message: err.Error(), Err: err}
}
