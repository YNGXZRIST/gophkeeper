package iface

import (
	"context"
	"gophkeeper/internal/client/auth"
)

// SessionStore loads and persists the current session; satisfied by the
// session repository.
type SessionStore interface {
	Get(ctx context.Context) (*auth.Session, error)
	Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error)
	Clear(ctx context.Context) error
}
