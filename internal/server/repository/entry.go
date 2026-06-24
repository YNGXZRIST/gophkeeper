package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
	"strings"
	"time"
)

const TableCard = "cards"
const TablePassword = "passwords"
const TableNote = "notes"

// Store is the persistence contract for one kind of encrypted entry — notes,
// passwords or cards. Every kind is served by its own EntryRepo bound to the
// matching table; the payload is opaque to the store and stays encrypted.
type Store interface {
	GetByUser(ctx context.Context, uid string, lastID string, limit, offset int) ([]*model.Entry, error)
	GetByID(ctx context.Context, uid string, id string) (*model.Entry, error)
	Create(ctx context.Context, uid string, secret *model.Entry) (*model.Entry, error)
	Update(ctx context.Context, uid string, secret *model.Entry) (*model.Entry, error)
	Delete(ctx context.Context, uid string, id string) error
	Changes(ctx context.Context, uid string, since time.Time) ([]*model.EntryChange, error)
}

// EntryRepo is a Store backed by a single table. Notes, passwords and cards
// reuse this implementation and differ only by the table name and the queries
// bound to it at construction.
type EntryRepo struct {
	repoBase
	label string

	listByUserQuery string
	getByIDQuery    string
	createQuery     string
	updateQuery     string
	deleteQuery     string
	changesQuery    string
}

// NewEntryRepo returns an EntryRepo that persists entries in the given table.
//
// table must be a trusted constant: it is interpolated directly into the SQL
// and is never escaped, so callers must not pass user input.
func NewEntryRepo(db *conn.DB, table string) *EntryRepo {
	return &EntryRepo{
		repoBase:        repoBase{db: db},
		label:           getRepositoryLabel(table),
		listByUserQuery: buildListByUserQuery(table),
		getByIDQuery:    buildGetByIDQuery(table),
		createQuery:     buildCreateQuery(table),
		updateQuery:     buildUpdateQuery(table),
		deleteQuery:     buildDeleteQuery(table),
		changesQuery:    buildChangesQuery(table),
	}
}

// GetByUser returns one page of the user's entries, ordered by id and starting
// after lastID for keyset pagination.
func (e *EntryRepo) GetByUser(ctx context.Context, uid string, lastID string, limit, offset int) ([]*model.Entry, error) {
	var cursor any
	if lastID != "" {
		cursor = lastID
	}
	return queryRows(ctx, e.q(ctx), e.label+".GetByUser", e.listByUserQuery, scanEntry, uid, cursor, limit, offset)
}

func scanEntry(s scanner) (*model.Entry, error) {
	var entry model.Entry
	if err := s.Scan(&entry.ID, &entry.UserID, &entry.Data, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetByID returns a single entry owned by the user, or the repository's
// not-found error when it is missing or belongs to someone else.
func (e *EntryRepo) GetByID(ctx context.Context, uid string, id string) (*model.Entry, error) {
	var entry model.Entry
	err := e.q(ctx).QueryRowContext(ctx, e.getByIDQuery, id, uid).
		Scan(&entry.ID, &entry.UserID, &entry.Data, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrEntryNotFound
		}
		return nil, labelerrors.NewLabelError(e.label+".GetByID", err)
	}
	return &entry, nil
}

// Create stores a new entry for the user, reviving a soft-deleted one when an
// entry with the same id already exists.
func (e *EntryRepo) Create(ctx context.Context, uid string, entry *model.Entry) (*model.Entry, error) {
	err := e.q(ctx).QueryRowContext(ctx, e.createQuery, entry.ID, uid, entry.Data).
		Scan(&entry.ID, &entry.UserID, &entry.Data, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrEntryNotFound
		}
		return nil, labelerrors.NewLabelError(e.label+".Create", err)
	}
	return entry, nil
}

// Update overwrites the entry's encrypted payload using optimistic versioning
// and reports ErrVersionConflict when the stored version has moved on.
func (e *EntryRepo) Update(ctx context.Context, uid string, entry *model.Entry) (*model.Entry, error) {
	err := e.q(ctx).QueryRowContext(ctx, e.updateQuery, entry.Data, entry.ID, uid, entry.Version).
		Scan(&entry.ID, &entry.UserID, &entry.Data, &entry.Version, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVersionConflict
		}
		return nil, labelerrors.NewLabelError(e.label+".Update", err)
	}
	return entry, nil
}

// Delete soft-deletes the user's entry by id and reports the not-found error
// when nothing was deleted.
func (e *EntryRepo) Delete(ctx context.Context, uid string, id string) error {
	res, err := e.q(ctx).ExecContext(ctx, e.deleteQuery, id, uid)
	if err != nil {
		return labelerrors.NewLabelError(e.label+".Delete.ExecContext", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return labelerrors.NewLabelError(e.label+".Delete.RowsAffected", err)
	}
	if affected == 0 {
		return model.ErrEntryNotFound
	}
	return nil
}

// Changes returns the user's entries, including tombstones, changed after since.
// A zero since returns the full history and is used for the initial sync.
func (e *EntryRepo) Changes(ctx context.Context, uid string, since time.Time) ([]*model.EntryChange, error) {
	var cursor any
	if !since.IsZero() {
		cursor = since
	}

	return queryRows(ctx, e.q(ctx), e.label+".Changes", e.changesQuery, scanEntryChange, uid, cursor)
}

func scanEntryChange(s scanner) (*model.EntryChange, error) {
	var ch model.EntryChange
	if err := s.Scan(&ch.ID, &ch.Data, &ch.Version, &ch.Deleted, &ch.UpdatedAt); err != nil {
		return nil, err
	}
	return &ch, nil
}

// getRepositoryLabel builds the error-label prefix for a table's repository.
func getRepositoryLabel(table string) string {
	return labelRepository + "." + strings.ToTitle(table)
}

// buildListByUserQuery renders the keyset-paginated listing query for table.
func buildListByUserQuery(table string) string {
	return fmt.Sprintf(`SELECT id, user_id, data, version, created_at, updated_at
FROM %s
WHERE user_id = $1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`, table)
}

// buildGetByIDQuery renders the single-entry lookup query for table.
func buildGetByIDQuery(table string) string {
	return fmt.Sprintf(`SELECT id, user_id, data, version, created_at, updated_at
FROM %s
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, table)
}

// buildCreateQuery renders the insert-or-revive upsert query for table.
func buildCreateQuery(table string) string {
	return fmt.Sprintf(`INSERT INTO %[1]s (id, user_id, data) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET data = excluded.data, version = %[1]s.version + 1, updated_at = now(), deleted_at = NULL
WHERE %[1]s.user_id = excluded.user_id
RETURNING id, user_id, data, version, created_at, updated_at`, table)
}

// buildUpdateQuery renders the optimistic-locking update query for table.
func buildUpdateQuery(table string) string {
	return fmt.Sprintf(`UPDATE %s
SET data = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, data, version, created_at, updated_at`, table)
}

// buildDeleteQuery renders the soft-delete query for table.
func buildDeleteQuery(table string) string {
	return fmt.Sprintf(`UPDATE %s
SET deleted_at = now(), version = version + 1, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, table)
}

// buildChangesQuery renders the incremental-sync query for table.
func buildChangesQuery(table string) string {
	return fmt.Sprintf(`SELECT id, data, version, deleted_at IS NOT NULL, updated_at
FROM %s
WHERE user_id = $1 AND ($2::timestamptz IS NULL OR updated_at > $2::timestamptz)
ORDER BY updated_at`, table)
}
