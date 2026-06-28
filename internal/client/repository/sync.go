package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SyncRow is a local row as seen by the sync engine, independent of entity type.
type SyncRow struct {
	ID          string
	Data        []byte
	Version     int64
	BaseVersion int64
	Dirty       bool
	Deleted     bool
}

// ConflictRow holds a local/server pair for a row awaiting conflict resolution.
type ConflictRow struct {
	ID            string
	Local         []byte
	Server        []byte
	ServerVersion int64
}

type scanner interface{ Scan(dest ...any) error }

// queryRows runs query and decodes every result row with scan. It centralizes
// the rows.Next/Scan/Err boilerplate shared by all list-style queries.
func queryRows[T any](ctx context.Context, db *sql.DB, query string, scan func(scanner) (T, error), args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

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

func scanConflictRow(s scanner) (ConflictRow, error) {
	var c ConflictRow
	var version sql.NullInt64
	if err := s.Scan(&c.ID, &c.Local, &c.Server, &version); err != nil {
		return ConflictRow{}, err
	}
	c.ServerVersion = version.Int64
	return c, nil
}

// syncRepo implements the shared sync/conflict persistence for an entity table.
// It is embedded by the per-entity repositories, which differ only in the table
// and payload column they operate on.
type syncRepo struct {
	db *sql.DB

	listDirtyQuery         string
	getRowQuery            string
	upsertQuery            string
	hardDeleteQuery        string
	markSyncedQuery        string
	markConflictQuery      string
	listConflictsQuery     string
	resolveKeepMineQuery   string
	resolveTakeServerQuery string
}

// newSyncRepo precomputes the queries for the given table and payload column.
// table and payloadCol MUST be trusted constants (never user input): they are
// interpolated into SQL and would otherwise allow injection.
func newSyncRepo(db *sql.DB, table, payloadCol string) syncRepo {
	return syncRepo{
		db: db,
		listDirtyQuery: fmt.Sprintf(
			`SELECT id, %[2]s, version, base_version, dirty, deleted FROM %[1]s WHERE dirty = 1 AND conflict = 0`,
			table, payloadCol),
		getRowQuery: fmt.Sprintf(
			`SELECT id, %[2]s, version, base_version, dirty, deleted FROM %[1]s WHERE id = ?`,
			table, payloadCol),
		upsertQuery: fmt.Sprintf(
			`INSERT INTO %[1]s (id, %[2]s, version, base_version, dirty, deleted) VALUES (?, ?, ?, ?, 0, 0)
ON CONFLICT(id) DO UPDATE SET %[2]s = excluded.%[2]s, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0`,
			table, payloadCol),
		hardDeleteQuery:   fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table),
		markSyncedQuery:   fmt.Sprintf(`UPDATE %s SET version = ?, base_version = ?, dirty = 0 WHERE id = ?`, table),
		markConflictQuery: fmt.Sprintf(`UPDATE %s SET conflict = 1, server_blob = ?, server_version = ? WHERE id = ?`, table),
		listConflictsQuery: fmt.Sprintf(
			`SELECT id, %[2]s, server_blob, server_version FROM %[1]s WHERE conflict = 1 AND server_blob IS NOT NULL`,
			table, payloadCol),
		resolveKeepMineQuery: fmt.Sprintf(
			`UPDATE %s SET base_version = server_version, dirty = 1, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`,
			table),
		resolveTakeServerQuery: fmt.Sprintf(
			`UPDATE %[1]s SET %[2]s = server_blob, version = server_version, base_version = server_version, dirty = 0, conflict = 0, server_blob = NULL, server_version = NULL WHERE id = ?`,
			table, payloadCol),
	}
}

func (r *syncRepo) ListDirty(ctx context.Context) ([]SyncRow, error) {
	return queryRows(ctx, r.db, r.listDirtyQuery, scanSyncRow)
}

func (r *syncRepo) GetRow(ctx context.Context, id string) (SyncRow, bool, error) {
	row, err := scanSyncRow(r.db.QueryRowContext(ctx, r.getRowQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return SyncRow{}, false, nil
	}
	if err != nil {
		return SyncRow{}, false, err
	}
	return row, true, nil
}

func (r *syncRepo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	_, err := r.db.ExecContext(ctx, r.upsertQuery, id, data, version, version)
	return err
}

func (r *syncRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.hardDeleteQuery, id)
	return err
}

func (r *syncRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	_, err := r.db.ExecContext(ctx, r.markSyncedQuery, version, version, id)
	return err
}

func (r *syncRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	_, err := r.db.ExecContext(ctx, r.markConflictQuery, serverBlob, serverVersion, id)
	return err
}

func (r *syncRepo) ListConflicts(ctx context.Context) ([]ConflictRow, error) {
	return queryRows(ctx, r.db, r.listConflictsQuery, scanConflictRow)
}

func (r *syncRepo) ResolveKeepMine(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.resolveKeepMineQuery, id)
	return err
}

func (r *syncRepo) ResolveTakeServer(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.resolveTakeServerQuery, id)
	return err
}
