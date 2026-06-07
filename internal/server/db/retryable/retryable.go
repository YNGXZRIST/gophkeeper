// Package retryable provides retry wrappers for DB operations.
package retryable

import (
	"context"
	"gophermart-loyalty/internal/gopherman/db/pgerrors"
	"gophermart-loyalty/internal/gopherman/errors/labelerrors"
	"time"
)

const maxRetries = 3

// RunWithRetryLabel is a label prefix for retryable errors.
const RunWithRetryLabel = "RunWithRetry"

// RunWithRetry executes operation with retry policy for retriable DB errors.
func RunWithRetry[T any](ctx context.Context, op func() (T, error)) (T, error) {
	var zero T
	var lastRes T
	var lastErr error
	sleepSeconds := 1
	classifier := pgerrors.NewPostgresErrorClassifier()
	for attempt := 0; attempt < maxRetries; attempt++ {
		lastRes, lastErr = op()
		if lastErr == nil {
			return lastRes, nil
		}
		if classifier.Classify(lastErr) == pgerrors.NonRetriable {
			return zero, labelerrors.NewLabelError(RunWithRetryLabel+".NonRetriable", pgerrors.NewPgError(lastErr))
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(time.Duration(sleepSeconds) * time.Second):
			sleepSeconds += 2
		}
	}
	if lastErr != nil {
		lastErr = labelerrors.NewLabelError(RunWithRetryLabel+".Retried", pgerrors.NewPgError(lastErr))
	}
	return lastRes, lastErr
}
