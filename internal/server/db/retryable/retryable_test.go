package retryable

import (
	"context"
	"errors"
	"testing"

	"gophkeeper/internal/server/db/pgerrors"
	"gophkeeper/internal/shared/errors/labelerrors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRunWithRetrySuccess(t *testing.T) {
	calls := 0
	got, err := RunWithRetry(context.Background(), func() (int, error) {
		calls++
		return 42, nil
	})
	if err != nil {
		t.Fatalf("RunWithRetry() error = %v", err)
	}
	if got != 42 {
		t.Fatalf("RunWithRetry() = %d, want 42", got)
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want 1", calls)
	}
}

func TestRunWithRetryNonRetriable(t *testing.T) {
	calls := 0
	nonRetriable := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	got, err := RunWithRetry(context.Background(), func() (int, error) {
		calls++
		return 0, nonRetriable
	})
	if err == nil {
		t.Fatal("RunWithRetry() error = nil, want non-retriable error")
	}
	if got != 0 {
		t.Fatalf("RunWithRetry() = %d, want zero value", got)
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want 1 (no retry)", calls)
	}
	var le labelerrors.LabelError
	if !errors.As(err, &le) {
		t.Fatalf("error is not LabelError: %T", err)
	}
	if le.Label != RunWithRetryLabel+".NonRetriable" {
		t.Fatalf("label = %q, want %q", le.Label, RunWithRetryLabel+".NonRetriable")
	}
}

func TestRunWithRetryContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	retriable := &pgconn.PgError{Code: pgerrcode.DeadlockDetected}
	_, err := RunWithRetry(ctx, func() (int, error) {
		calls++
		return 0, retriable
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunWithRetry() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want 1 before ctx cancellation", calls)
	}
}

func TestRunWithRetryExhausted(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: relies on hardcoded retry backoff (~4s)")
	}
	calls := 0
	retriable := &pgconn.PgError{Code: pgerrcode.SerializationFailure}
	_, err := RunWithRetry(context.Background(), func() (int, error) {
		calls++
		return 0, retriable
	})
	if err == nil {
		t.Fatal("RunWithRetry() error = nil, want retried error")
	}
	if calls != 3 {
		t.Fatalf("op called %d times, want 3", calls)
	}
	var le labelerrors.LabelError
	if !errors.As(err, &le) {
		t.Fatalf("error is not LabelError: %T", err)
	}
	if le.Label != RunWithRetryLabel+".Retried" {
		t.Fatalf("label = %q, want %q", le.Label, RunWithRetryLabel+".Retried")
	}
}

func TestRunWithRetryClassifierUsesRetriable(t *testing.T) {
	c := pgerrors.NewPostgresErrorClassifier()
	if c.Classify(&pgconn.PgError{Code: pgerrcode.DeadlockDetected}) != pgerrors.Retriable {
		t.Fatal("precondition: deadlock should be retriable")
	}
}
