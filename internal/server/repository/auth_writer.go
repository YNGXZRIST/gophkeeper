package repository

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/server/model"
)

// AuthWriter runs the transactional refresh-token flows for auth: creating a
// user with its first token (registration) and rotating the token (login), so
// each flow's DB writes are all-or-nothing.
type AuthWriter struct {
	mgr     *trmanager.Manager
	rep     Repositories
	refresh *RefreshIssuer
}

// NewAuthWriter constructs a transactional auth writer.
func NewAuthWriter(mgr *trmanager.Manager, rep Repositories, refresh *RefreshIssuer) *AuthWriter {
	return &AuthWriter{mgr: mgr, rep: rep, refresh: refresh}
}

// Register persists the user and a freshly generated refresh token atomically,
// returning the created user and the plaintext refresh token.
func (w *AuthWriter) Register(ctx context.Context, u model.User) (*model.User, string, error) {
	var (
		created *model.User
		plain   string
	)
	err := w.mgr.WithinTx(ctx, nil, func(ctx context.Context) error {
		var err error
		if created, err = w.rep.User.Create(ctx, u); err != nil {
			return err
		}
		plain, err = w.refresh.Issue(ctx, created.ID)
		return err
	})
	if err != nil {
		return nil, "", fmt.Errorf("register: %w", err)
	}
	return created, plain, nil
}

// IssueRefresh issues a new refresh token for the user without touching tokens
// already issued to other devices, so the same account can stay logged in on
// several devices at once. It is a single insert, hence no transaction.
func (w *AuthWriter) IssueRefresh(ctx context.Context, userID string) (string, error) {
	return w.refresh.Issue(ctx, userID)
}
