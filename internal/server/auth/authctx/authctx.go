// Package authctx carries the authenticated user identity in the request context.
package authctx

import "context"

type ctxKey struct{}

// ContextWithUserID returns ctx carrying the authenticated user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserIDFromContext returns the authenticated user ID from ctx, if present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(ctxKey{}).(string)
	return userID, ok && userID != ""
}
