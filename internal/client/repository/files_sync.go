package repository

import (
	"context"
	"database/sql"
	"errors"
)

const filesListDirtyQuery = `SELECT id, meta, version, base_version, dirty, deleted FROM files WHERE dirty = 1 AND conflict = 0`
const filesGetRowQuery = `SELECT id, meta, version, base_version, dirty, deleted FROM files WHERE id = ?`
const filesUpsertQuery = `INSERT INTO files (id, meta, version, base_version, dirty, deleted) VALUES (?, ?, ?, ?, 0, 0)
ON CONFLICT(id) DO UPDATE SET meta = excluded.meta, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0`
const filesHardDeleteQuery = `DELETE FROM files WHERE id = ?`
const filesMarkSyncedQuery = `UPDATE files SET version = ?, base_version = ?, dirty = 0 WHERE id = ?`
const filesMarkConflictQuery = `UPDATE files SET conflict = 1, server_blob = ?, server_version = ? WHERE id = ?`

func (r *FilesRepo) ListDirty(ctx context.Context) ([]SyncRow, error) {
	rows, err := r.db.QueryContext(ctx, filesListDirtyQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SyncRow
	for rows.Next() {
		row, err := scanSyncRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FilesRepo) GetRow(ctx context.Context, id string) (SyncRow, bool, error) {
	row, err := scanSyncRow(r.db.QueryRowContext(ctx, filesGetRowQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SyncRow{}, false, nil
	}
	if err != nil {
		return SyncRow{}, false, err
	}
	return row, true, nil
}

func (r *FilesRepo) Upsert(ctx context.Context, id string, meta []byte, version int64) error {
	_, err := r.db.ExecContext(ctx, filesUpsertQuery, id, meta, version, version)
	return err
}

func (r *FilesRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, filesHardDeleteQuery, id)
	return err
}

func (r *FilesRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	_, err := r.db.ExecContext(ctx, filesMarkSyncedQuery, version, version, id)
	return err
}

func (r *FilesRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	_, err := r.db.ExecContext(ctx, filesMarkConflictQuery, serverBlob, serverVersion, id)
	return err
}

const filesListConflictsQuery = `SELECT id, meta, server_blob, server_version FROM files WHERE conflict = 1 AND server_blob IS NOT NULL`
const filesResolveKeepMineQuery = `UPDATE files SET base_version = server_version, dirty = 1, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`
const filesResolveTakeServerQuery = `UPDATE files SET meta = server_blob, version = server_version, base_version = server_version, dirty = 0, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`

func (r *FilesRepo) ListConflicts(ctx context.Context) ([]ConflictRow, error) {
	rows, err := r.db.QueryContext(ctx, filesListConflictsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ConflictRow
	for rows.Next() {
		var c ConflictRow
		var version sql.NullInt64
		if err := rows.Scan(&c.ID, &c.Local, &c.Server, &version); err != nil {
			return nil, err
		}
		c.ServerVersion = version.Int64
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FilesRepo) ResolveKeepMine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, filesResolveKeepMineQuery, id)
	return err
}

func (r *FilesRepo) ResolveTakeServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, filesResolveTakeServerQuery, id)
	return err
}
