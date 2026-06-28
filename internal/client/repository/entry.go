package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Table names for the blob-based entities. They are trusted constants and the
// only thing that distinguishes one EntryRepo instance from another.
const (
	TableNote     = "notes"
	TableCard     = "cards"
	TablePassword = "passwords"
)

// EntryRow is a decoded local row for the blob-based entities (notes, cards,
// passwords), which share an identical schema and differ only by table.
type EntryRow struct {
	ID      string
	Data    []byte
	Version int64
}

// NoteRow, CardRow and PasswordRow are aliases for the shared EntryRow; kept
// for call-site clarity at the TUI layer.
type (
	NoteRow     = EntryRow
	CardRow     = EntryRow
	PasswordRow = EntryRow
)

// EntryRepo persists one blob-based entity in a single table. Notes, cards and
// passwords reuse this type and differ only by the table bound at construction.
type EntryRepo struct {
	syncRepo

	listQuery   string
	insertQuery string
	updateQuery string
	deleteQuery string
}

// NewEntryRepo precomputes the queries for the given table. table MUST be a
// trusted constant (never user input): it is interpolated into SQL.
func NewEntryRepo(db *sql.DB, table string) *EntryRepo {
	return &EntryRepo{
		syncRepo: newSyncRepo(db, table, "data"),
		listQuery: fmt.Sprintf(
			`SELECT id, data, version FROM %s
WHERE deleted = 0 AND (? = '' OR id > ?)
ORDER BY id LIMIT ?`, table),
		insertQuery: fmt.Sprintf(
			`INSERT INTO %s (id, data, version, dirty, base_version) VALUES (?, ?, 1, 1, 0)`, table),
		updateQuery: fmt.Sprintf(
			`UPDATE %s SET data = ?, version = version + 1, dirty = 1,
base_version = CASE WHEN conflict = 1 THEN COALESCE(server_version, base_version) ELSE base_version END,
conflict = 0, server_blob = NULL, server_version = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, table),
		deleteQuery: fmt.Sprintf(
			`UPDATE %s SET deleted = 1, dirty = 1, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, table),
	}
}

func scanEntryRow(s scanner) (EntryRow, error) {
	var e EntryRow
	err := s.Scan(&e.ID, &e.Data, &e.Version)
	return e, err
}

func (r *EntryRepo) List(ctx context.Context, lastID string, limit int) ([]EntryRow, error) {
	return queryRows(ctx, r.db, r.listQuery, scanEntryRow, lastID, lastID, limit)
}

func (r *EntryRepo) Create(ctx context.Context, data []byte) (EntryRow, error) {
	id := uuid.NewString()
	if _, err := r.db.ExecContext(ctx, r.insertQuery, id, data); err != nil {
		return EntryRow{}, err
	}
	return EntryRow{ID: id, Data: data, Version: 1}, nil
}

func (r *EntryRepo) Update(ctx context.Context, id string, data []byte) error {
	_, err := r.db.ExecContext(ctx, r.updateQuery, data, id)
	return err
}

func (r *EntryRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.deleteQuery, id)
	return err
}
