package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type PasswordsRepo struct {
	repoBase
}

func NewPasswordsRepo(conn *sql.DB) *PasswordsRepo {
	return &PasswordsRepo{repoBase: repoBase{db: conn}}
}

type PasswordRow struct {
	ID      string
	Data    []byte
	Version int64
}

const passwordsListQuery = `SELECT id, data, version FROM passwords
WHERE deleted = 0 AND (? = '' OR id > ?)
ORDER BY id LIMIT ?`
const passwordsInsertQuery = `INSERT INTO passwords (id, data, version, dirty, base_version) VALUES (?, ?, 1, 1, 0)`
const passwordsUpdateQuery = `UPDATE passwords SET data = ?, version = version + 1, dirty = 1,
base_version = CASE WHEN conflict = 1 THEN COALESCE(server_version, base_version) ELSE base_version END,
conflict = 0, server_blob = NULL, server_version = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
const passwordsDeleteQuery = `UPDATE passwords SET deleted = 1, dirty = 1, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

func (r *PasswordsRepo) List(ctx context.Context, lastID string, limit int) ([]PasswordRow, error) {
	rows, err := r.db.QueryContext(ctx, passwordsListQuery, lastID, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PasswordRow
	for rows.Next() {
		var p PasswordRow
		if err := rows.Scan(&p.ID, &p.Data, &p.Version); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PasswordsRepo) Create(ctx context.Context, data []byte) (PasswordRow, error) {
	id := uuid.NewString()
	if _, err := r.db.ExecContext(ctx, passwordsInsertQuery, id, data); err != nil {
		return PasswordRow{}, err
	}
	return PasswordRow{ID: id, Data: data, Version: 1}, nil
}

func (r *PasswordsRepo) Update(ctx context.Context, id string, data []byte) error {
	_, err := r.db.ExecContext(ctx, passwordsUpdateQuery, data, id)
	return err
}

func (r *PasswordsRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, passwordsDeleteQuery, id)
	return err
}
