package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
)

type RefreshTokenRepo struct {
	repoBase
}

const (
	RefreshTokenCreateQuery = `INSERT INTO refresh_tokens(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at`

	RefreshTokenGetByHashQuery = `SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens WHERE token_hash = $1`

	RefreshTokenDeleteByHashQuery = `DELETE FROM refresh_tokens WHERE token_hash = $1`
)

func NewRefreshTokenRepo(db *conn.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{repoBase: repoBase{db: db}}
}

// Create stores a refresh token and returns the persisted row.
func (r *RefreshTokenRepo) Create(ctx context.Context, rt model.RefreshToken) (*model.RefreshToken, error) {
	err := r.repoBase.q(ctx).QueryRowContext(ctx,
		RefreshTokenCreateQuery,
		rt.UserID, rt.TokenHash, rt.ExpiresAt,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".RefreshToken.Create", err)
	}
	return &rt, nil
}

// GetByHash returns the refresh token matching the given hash.
func (r *RefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.repoBase.q(ctx).QueryRowContext(ctx,
		RefreshTokenGetByHashQuery, tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".RefreshToken.GetByHash", err)
	}
	return &rt, nil
}

// DeleteByHash removes a refresh token, e.g. on logout or rotation.
func (r *RefreshTokenRepo) DeleteByHash(ctx context.Context, tokenHash string) error {
	if _, err := r.repoBase.q(ctx).ExecContext(ctx, RefreshTokenDeleteByHashQuery, tokenHash); err != nil {
		return labelerrors.NewLabelError(labelRepository+".RefreshToken.DeleteByHash", err)
	}
	return nil
}
