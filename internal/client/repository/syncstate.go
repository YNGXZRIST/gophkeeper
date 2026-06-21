package repository

import (
	"context"
	"database/sql"
	"errors"
)

type SyncStateRepo struct {
	repoBase
}

func NewSyncStateRepo(conn *sql.DB) *SyncStateRepo {
	return &SyncStateRepo{repoBase: repoBase{db: conn}}
}

const syncStateGetQuery = `SELECT cursor FROM sync_state WHERE entity = ?`
const syncStateSetQuery = `INSERT INTO sync_state (entity, cursor) VALUES (?, ?)
ON CONFLICT(entity) DO UPDATE SET cursor = excluded.cursor`

func (r *SyncStateRepo) Cursor(ctx context.Context, entity string) (string, error) {
	var cursor string
	err := r.db.QueryRowContext(ctx, syncStateGetQuery, entity).Scan(&cursor)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return cursor, nil
}

func (r *SyncStateRepo) SetCursor(ctx context.Context, entity, cursor string) error {
	_, err := r.db.ExecContext(ctx, syncStateSetQuery, entity, cursor)
	return err
}
