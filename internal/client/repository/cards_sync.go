package repository

import (
	"context"
	"database/sql"
	"errors"
)

const cardsListDirtyQuery = `SELECT id, data, version, base_version, dirty, deleted FROM cards WHERE dirty = 1 AND conflict = 0`
const cardsGetRowQuery = `SELECT id, data, version, base_version, dirty, deleted FROM cards WHERE id = ?`
const cardsUpsertQuery = `INSERT INTO cards (id, data, version, base_version, dirty, deleted) VALUES (?, ?, ?, ?, 0, 0)
ON CONFLICT(id) DO UPDATE SET data = excluded.data, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0`
const cardsHardDeleteQuery = `DELETE FROM cards WHERE id = ?`
const cardsMarkSyncedQuery = `UPDATE cards SET version = ?, base_version = ?, dirty = 0 WHERE id = ?`
const cardsMarkConflictQuery = `UPDATE cards SET conflict = 1, server_blob = ?, server_version = ? WHERE id = ?`

func (r *CardsRepo) ListDirty(ctx context.Context) ([]SyncRow, error) {
	rows, err := r.db.QueryContext(ctx, cardsListDirtyQuery)
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

func (r *CardsRepo) GetRow(ctx context.Context, id string) (SyncRow, bool, error) {
	row, err := scanSyncRow(r.db.QueryRowContext(ctx, cardsGetRowQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SyncRow{}, false, nil
	}
	if err != nil {
		return SyncRow{}, false, err
	}
	return row, true, nil
}

func (r *CardsRepo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	_, err := r.db.ExecContext(ctx, cardsUpsertQuery, id, data, version, version)
	return err
}

func (r *CardsRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, cardsHardDeleteQuery, id)
	return err
}

func (r *CardsRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	_, err := r.db.ExecContext(ctx, cardsMarkSyncedQuery, version, version, id)
	return err
}

func (r *CardsRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	_, err := r.db.ExecContext(ctx, cardsMarkConflictQuery, serverBlob, serverVersion, id)
	return err
}

const cardsListConflictsQuery = `SELECT id, data, server_blob, server_version FROM cards WHERE conflict = 1 AND server_blob IS NOT NULL`
const cardsResolveKeepMineQuery = `UPDATE cards SET base_version = server_version, dirty = 1, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`
const cardsResolveTakeServerQuery = `UPDATE cards SET data = server_blob, version = server_version, base_version = server_version, dirty = 0, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`

func (r *CardsRepo) ListConflicts(ctx context.Context) ([]ConflictRow, error) {
	rows, err := r.db.QueryContext(ctx, cardsListConflictsQuery)
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

func (r *CardsRepo) ResolveKeepMine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, cardsResolveKeepMineQuery, id)
	return err
}

func (r *CardsRepo) ResolveTakeServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, cardsResolveTakeServerQuery, id)
	return err
}
