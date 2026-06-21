package repository

import (
	"context"
	"database/sql"
	"errors"
)

type SyncRow struct {
	ID          string
	Data        []byte
	Version     int64
	BaseVersion int64
	Dirty       bool
	Deleted     bool
}

const notesListDirtyQuery = `SELECT id, data, version, base_version, dirty, deleted FROM notes WHERE dirty = 1 AND conflict = 0`
const notesGetRowQuery = `SELECT id, data, version, base_version, dirty, deleted FROM notes WHERE id = ?`
const notesUpsertQuery = `INSERT INTO notes (id, data, version, base_version, dirty, deleted) VALUES (?, ?, ?, ?, 0, 0)
ON CONFLICT(id) DO UPDATE SET data = excluded.data, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0`
const notesHardDeleteQuery = `DELETE FROM notes WHERE id = ?`
const notesMarkSyncedQuery = `UPDATE notes SET version = ?, base_version = ?, dirty = 0 WHERE id = ?`
const notesMarkConflictQuery = `UPDATE notes SET conflict = 1, server_blob = ?, server_version = ? WHERE id = ?`

type scanner interface{ Scan(dest ...any) error }

func scanSyncRow(s scanner) (SyncRow, error) {
	var row SyncRow
	var dirty, deleted int
	if err := s.Scan(&row.ID, &row.Data, &row.Version, &row.BaseVersion, &dirty, &deleted); err != nil {
		return SyncRow{}, err
	}
	row.Dirty = dirty != 0
	row.Deleted = deleted != 0
	return row, nil
}

func (r *NotesRepo) ListDirty(ctx context.Context) ([]SyncRow, error) {
	rows, err := r.db.QueryContext(ctx, notesListDirtyQuery)
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

func (r *NotesRepo) GetRow(ctx context.Context, id string) (SyncRow, bool, error) {
	row, err := scanSyncRow(r.db.QueryRowContext(ctx, notesGetRowQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SyncRow{}, false, nil
	}
	if err != nil {
		return SyncRow{}, false, err
	}
	return row, true, nil
}

func (r *NotesRepo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	_, err := r.db.ExecContext(ctx, notesUpsertQuery, id, data, version, version)
	return err
}

func (r *NotesRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, notesHardDeleteQuery, id)
	return err
}

func (r *NotesRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	_, err := r.db.ExecContext(ctx, notesMarkSyncedQuery, version, version, id)
	return err
}

func (r *NotesRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	_, err := r.db.ExecContext(ctx, notesMarkConflictQuery, serverBlob, serverVersion, id)
	return err
}

type ConflictRow struct {
	ID            string
	Local         []byte
	Server        []byte
	ServerVersion int64
}

const notesListConflictsQuery = `SELECT id, data, server_blob, server_version FROM notes WHERE conflict = 1 AND server_blob IS NOT NULL`
const notesResolveKeepMineQuery = `UPDATE notes SET base_version = server_version, dirty = 1, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`
const notesResolveTakeServerQuery = `UPDATE notes SET data = server_blob, version = server_version, base_version = server_version, dirty = 0, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`

func (r *NotesRepo) ListConflicts(ctx context.Context) ([]ConflictRow, error) {
	rows, err := r.db.QueryContext(ctx, notesListConflictsQuery)
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

func (r *NotesRepo) ResolveKeepMine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, notesResolveKeepMineQuery, id)
	return err
}

func (r *NotesRepo) ResolveTakeServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, notesResolveTakeServerQuery, id)
	return err
}
