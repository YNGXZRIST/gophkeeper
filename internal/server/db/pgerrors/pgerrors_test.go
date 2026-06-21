package pgerrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassify(t *testing.T) {
	c := NewPostgresErrorClassifier()

	tests := []struct {
		name string
		err  error
		want PGErrorClassification
	}{
		{name: "nil", err: nil, want: NonRetriable},
		{name: "non-pg error", err: errors.New("boom"), want: NonRetriable},
		{name: "connection failure", err: &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, want: Retriable},
		{name: "connection exception", err: &pgconn.PgError{Code: pgerrcode.ConnectionException}, want: Retriable},
		{name: "connection does not exist", err: &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist}, want: Retriable},
		{name: "transaction rollback", err: &pgconn.PgError{Code: pgerrcode.TransactionRollback}, want: Retriable},
		{name: "serialization failure", err: &pgconn.PgError{Code: pgerrcode.SerializationFailure}, want: Retriable},
		{name: "deadlock detected", err: &pgconn.PgError{Code: pgerrcode.DeadlockDetected}, want: Retriable},
		{name: "cannot connect now", err: &pgconn.PgError{Code: pgerrcode.CannotConnectNow}, want: Retriable},
		{name: "unique violation", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: NonRetriable},
		{name: "foreign key violation", err: &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}, want: NonRetriable},
		{name: "not null violation", err: &pgconn.PgError{Code: pgerrcode.NotNullViolation}, want: NonRetriable},
		{name: "check violation", err: &pgconn.PgError{Code: pgerrcode.CheckViolation}, want: NonRetriable},
		{name: "integrity violation", err: &pgconn.PgError{Code: pgerrcode.IntegrityConstraintViolation}, want: NonRetriable},
		{name: "restrict violation", err: &pgconn.PgError{Code: pgerrcode.RestrictViolation}, want: NonRetriable},
		{name: "data exception", err: &pgconn.PgError{Code: pgerrcode.DataException}, want: NonRetriable},
		{name: "null value not allowed", err: &pgconn.PgError{Code: pgerrcode.NullValueNotAllowedDataException}, want: NonRetriable},
		{name: "syntax error", err: &pgconn.PgError{Code: pgerrcode.SyntaxError}, want: NonRetriable},
		{name: "undefined table", err: &pgconn.PgError{Code: pgerrcode.UndefinedTable}, want: NonRetriable},
		{name: "undefined column", err: &pgconn.PgError{Code: pgerrcode.UndefinedColumn}, want: NonRetriable},
		{name: "undefined function", err: &pgconn.PgError{Code: pgerrcode.UndefinedFunction}, want: NonRetriable},
		{name: "syntax or access rule", err: &pgconn.PgError{Code: pgerrcode.SyntaxErrorOrAccessRuleViolation}, want: NonRetriable},
		{name: "unknown code", err: &pgconn.PgError{Code: "99999"}, want: NonRetriable},
		{name: "wrapped retriable pg error", err: fmt.Errorf("wrap: %w", &pgconn.PgError{Code: pgerrcode.DeadlockDetected}), want: Retriable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.Classify(tt.err); got != tt.want {
				t.Fatalf("Classify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPgError(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if err := NewPgError(nil); err != nil {
			t.Fatalf("NewPgError(nil) = %v, want nil", err)
		}
	})

	t.Run("pg error keeps code and message", func(t *testing.T) {
		src := &pgconn.PgError{Code: pgerrcode.UniqueViolation, Message: "dup"}
		err := NewPgError(src)
		var pe *PgErrors
		if !errors.As(err, &pe) {
			t.Fatalf("NewPgError did not return *PgErrors: %T", err)
		}
		if pe.Code != pgerrcode.UniqueViolation {
			t.Errorf("Code = %q, want %q", pe.Code, pgerrcode.UniqueViolation)
		}
		if pe.Message != "dup" {
			t.Errorf("Message = %q, want dup", pe.Message)
		}
		if !errors.Is(err, src) {
			t.Error("wrapped error not unwrappable to source")
		}
	})

	t.Run("non-pg error stores message", func(t *testing.T) {
		src := errors.New("plain")
		err := NewPgError(src)
		var pe *PgErrors
		if !errors.As(err, &pe) {
			t.Fatalf("NewPgError did not return *PgErrors: %T", err)
		}
		if pe.Code != "" {
			t.Errorf("Code = %q, want empty", pe.Code)
		}
		if pe.Message != "plain" {
			t.Errorf("Message = %q, want plain", pe.Message)
		}
	})
}

func TestPgErrorsError(t *testing.T) {
	withErr := PgErrors{Code: "23505", Message: "dup", Err: errors.New("inner")}
	if got := withErr.Error(); got != "code=23505 dup: inner" {
		t.Errorf("Error() = %q", got)
	}

	noErr := PgErrors{Code: "42P01", Message: "missing"}
	if got := noErr.Error(); got != "code=42P01 missing" {
		t.Errorf("Error() = %q", got)
	}
}

func TestPgErrorsUnwrap(t *testing.T) {
	inner := errors.New("inner")
	e := PgErrors{Err: inner}
	if !errors.Is(e, inner) {
		t.Fatal("Unwrap did not expose inner error")
	}
}
