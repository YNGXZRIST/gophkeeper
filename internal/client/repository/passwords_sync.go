package repository

import (
	"context"
	"database/sql"
	"errors"
)

const passwordsListDirtyQuery = `SELECT id, data, version, base_version, dirty, deleted FROM passwords WHERE dirty = 1 AND conflict = 0`
const passwordsGetRowQuery = `SELECT id, data, version, base_version, dirty, deleted FROM passwords WHERE id = ?`
const passwordsUpsertQuery = `INSERT INTO passwords (id, data, version, base_version, dirty, deleted) VALUES (?, ?, ?, ?, 0, 0)
ON CONFLICT(id) DO UPDATE SET data = excluded.data, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0`
const passwordsHardDeleteQuery = `DELETE FROM passwords WHERE id = ?`
const passwordsMarkSyncedQuery = `UPDATE passwords SET version = ?, base_version = ?, dirty = 0 WHERE id = ?`
const passwordsMarkConflictQuery = `UPDATE passwords SET conflict = 1, server_blob = ?, server_version = ? WHERE id = ?`

func (r *PasswordsRepo) ListDirty(ctx context.Context) ([]SyncRow, error) {
	rows, err := r.db.QueryContext(ctx, passwordsListDirtyQuery)
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

func (r *PasswordsRepo) GetRow(ctx context.Context, id string) (SyncRow, bool, error) {
	row, err := scanSyncRow(r.db.QueryRowContext(ctx, passwordsGetRowQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SyncRow{}, false, nil
	}
	if err != nil {
		return SyncRow{}, false, err
	}
	return row, true, nil
}

func (r *PasswordsRepo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	_, err := r.db.ExecContext(ctx, passwordsUpsertQuery, id, data, version, version)
	return err
}

func (r *PasswordsRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, passwordsHardDeleteQuery, id)
	return err
}

func (r *PasswordsRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	_, err := r.db.ExecContext(ctx, passwordsMarkSyncedQuery, version, version, id)
	return err
}

func (r *PasswordsRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	_, err := r.db.ExecContext(ctx, passwordsMarkConflictQuery, serverBlob, serverVersion, id)
	return err
}

const passwordsListConflictsQuery = `SELECT id, data, server_blob, server_version FROM passwords WHERE conflict = 1 AND server_blob IS NOT NULL`
const passwordsResolveKeepMineQuery = `UPDATE passwords SET base_version = server_version, dirty = 1, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`
const passwordsResolveTakeServerQuery = `UPDATE passwords SET data = server_blob, version = server_version, base_version = server_version, dirty = 0, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`

func (r *PasswordsRepo) ListConflicts(ctx context.Context) ([]ConflictRow, error) {
	rows, err := r.db.QueryContext(ctx, passwordsListConflictsQuery)
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

func (r *PasswordsRepo) ResolveKeepMine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, passwordsResolveKeepMineQuery, id)
	return err
}

func (r *PasswordsRepo) ResolveTakeServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, passwordsResolveTakeServerQuery, id)
	return err
}
