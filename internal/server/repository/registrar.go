package repository

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/auth/refresh"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/server/model"
	"time"
)

// DefaultRefreshTTL is how long an issued refresh token stays valid.
const DefaultRefreshTTL = 30 * 24 * time.Hour

// Registrar creates a user and its initial refresh token within a single
// transaction, so neither is persisted without the other.
type Registrar struct {
	mgr           *trmanager.Manager
	rep           Repositories
	refreshSecret []byte
	refreshTTL    time.Duration
}

// NewRegistrar constructs a transactional registrar.
func NewRegistrar(mgr *trmanager.Manager, rep Repositories, refreshSecret []byte, refreshTTL time.Duration) *Registrar {
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}
	return &Registrar{mgr: mgr, rep: rep, refreshSecret: refreshSecret, refreshTTL: refreshTTL}
}

// Register persists the user and a freshly generated refresh token atomically,
// returning the created user and the plaintext refresh token to hand to the client.
func (r *Registrar) Register(ctx context.Context, u model.User) (*model.User, string, error) {
	var (
		created *model.User
		plain   string
	)
	err := r.mgr.WithinTx(ctx, nil, func(ctx context.Context) error {
		var err error
		if created, err = r.rep.User.Create(ctx, u); err != nil {
			return err
		}

		if plain, err = refresh.Generate(); err != nil {
			return fmt.Errorf("generate refresh: %w", err)
		}
		_, err = r.rep.RefreshToken.Create(ctx, model.RefreshToken{
			UserID:    created.ID,
			TokenHash: refresh.Hash(plain, r.refreshSecret),
			ExpiresAt: time.Now().Add(r.refreshTTL),
		})
		return err
	})
	if err != nil {
		return nil, "", fmt.Errorf("register: %w", err)
	}
	return created, plain, nil
}
