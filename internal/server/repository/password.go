package repository

import (
	"context"
	"database/sql"
	"errors"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
	"time"
)

type PassRepo struct {
	repoBase
}

const PasswordListByUserQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM passwords
WHERE user_id = $1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`

const PasswordGetByIDQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM passwords
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const PasswordCreateQuery = `INSERT INTO passwords(id, user_id, data) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET data = excluded.data, version = passwords.version + 1, updated_at = now(), deleted_at = NULL
WHERE passwords.user_id = excluded.user_id
RETURNING id, user_id, data, version, created_at, updated_at`

const PasswordUpdateQuery = `UPDATE passwords
SET data = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, data, version, created_at, updated_at`

const PasswordDeleteQuery = `UPDATE passwords
SET deleted_at = now(), version = version + 1, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const PasswordChangesQuery = `SELECT id, data, version, deleted_at IS NOT NULL, updated_at
FROM passwords
WHERE user_id = $1 AND ($2::timestamptz IS NULL OR updated_at > $2::timestamptz)
ORDER BY updated_at`

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
	err := p.q(ctx).QueryRowContext(ctx, PasswordCreateQuery, pass.ID, user.ID, pass.Data).
		Scan(&pass.ID, &pass.UserID, &pass.Data, &pass.Version, &pass.CreatedAt, &pass.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPasswordNotFound
		}
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

// Changes returns all the user's passwords (including tombstones) updated after since.
func (p PassRepo) Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.PasswordChange, error) {
	var cursor any
	if !since.IsZero() {
		cursor = since
	}

	rows, err := p.q(ctx).QueryContext(ctx, PasswordChangesQuery, user.ID, cursor)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Password.Changes.Query", err)
	}
	defer rows.Close()

	var changes []*model.PasswordChange
	for rows.Next() {
		var ch model.PasswordChange
		if err := rows.Scan(&ch.ID, &ch.Data, &ch.Version, &ch.Deleted, &ch.UpdatedAt); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Password.Changes.Scan", err)
		}
		changes = append(changes, &ch)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Password.Changes.Rows", err)
	}
	return changes, nil
}
