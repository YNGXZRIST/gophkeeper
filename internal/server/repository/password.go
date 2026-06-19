package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
)

type PassRepo struct {
	repoBase
}

const PasswordListByUserQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM passwords
WHERE user_id = $1 AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`

const PasswordGetByIDQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM passwords
WHERE id = $1 AND user_id = $2`

const PasswordCreateQuery = `INSERT INTO passwords(user_id, data) VALUES ($1, $2)
RETURNING id, user_id, data, version, created_at, updated_at`

const PasswordUpdateQuery = `UPDATE passwords
SET data = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, data, version, created_at, updated_at`

const PasswordDeleteQuery = `DELETE FROM passwords WHERE id = $1 AND user_id = $2`

func NewPasswordRepo(db *conn.DB) *PassRepo {
	return &PassRepo{repoBase: repoBase{db: db}}
}

// GetByUser returns one chunk of the user's encrypted passwords.
func (p PassRepo) GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Password, error) {
	var cursor any
	if lastID != "" {
		cursor = lastID
	}

	rows, err := p.q(ctx).QueryContext(ctx, PasswordListByUserQuery, user.ID, cursor, limit, offset)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Password.GetByUser.Query", err)
	}
	defer rows.Close()

	var passwords []*model.Password
	for rows.Next() {
		var pass model.Password
		if err := rows.Scan(
			&pass.ID, &pass.UserID, &pass.Data, &pass.Version, &pass.CreatedAt, &pass.UpdatedAt,
		); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Password.GetByUser.Scan", err)
		}
		passwords = append(passwords, &pass)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Password.GetByUser.Rows", err)
	}
	return passwords, nil
}

// GetByID returns a single encrypted password owned by the user.
func (p PassRepo) GetByID(ctx context.Context, user *model.User, id string) (*model.Password, error) {
	var pass model.Password
	err := p.q(ctx).QueryRowContext(ctx, PasswordGetByIDQuery, id, user.ID).
		Scan(&pass.ID, &pass.UserID, &pass.Data, &pass.Version, &pass.CreatedAt, &pass.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPasswordNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Password.GetByID", err)
	}
	return &pass, nil
}

// Create stores a new encrypted password for the user.
func (p PassRepo) Create(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error) {
	err := p.q(ctx).QueryRowContext(ctx, PasswordCreateQuery, user.ID, pass.Data).
		Scan(&pass.ID, &pass.UserID, &pass.Data, &pass.Version, &pass.CreatedAt, &pass.UpdatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Password.Create", err)
	}
	return pass, nil
}

// Update overwrites the password's encrypted payload using optimistic versioning.
func (p PassRepo) Update(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error) {
	err := p.q(ctx).QueryRowContext(ctx, PasswordUpdateQuery, pass.Data, pass.ID, user.ID, pass.Version).
		Scan(&pass.ID, &pass.UserID, &pass.Data, &pass.Version, &pass.CreatedAt, &pass.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVersionConflict
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Password.Update", err)
	}
	return pass, nil
}

// Delete removes a password owned by the user.
func (p PassRepo) Delete(ctx context.Context, user *model.User, id string) error {
	res, err := p.q(ctx).ExecContext(ctx, PasswordDeleteQuery, id, user.ID)
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Password.Delete", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Password.Delete.RowsAffected", err)
	}
	if affected == 0 {
		return model.ErrPasswordNotFound
	}
	return nil
}
