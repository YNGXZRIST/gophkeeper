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

type NoteRepo struct {
	repoBase
}

func NewNoteRepo(db *conn.DB) *NoteRepo {
	return &NoteRepo{repoBase: repoBase{db: db}}
}

const NoteListByUserQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM notes
WHERE user_id = $1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`

const NoteGetByIDQuery = `SELECT id, user_id, data, version, created_at, updated_at
FROM notes
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const NoteCreateQuery = `INSERT INTO notes(id, user_id, data) VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET data = excluded.data, version = notes.version + 1, updated_at = now(), deleted_at = NULL
WHERE notes.user_id = excluded.user_id
RETURNING id, user_id, data, version, created_at, updated_at`

const NoteUpdateQuery = `UPDATE notes
SET data = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, data, version, created_at, updated_at`

const NoteDeleteQuery = `UPDATE notes
SET deleted_at = now(), version = version + 1, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const NoteChangesQuery = `SELECT id, data, version, deleted_at IS NOT NULL, updated_at
FROM notes
WHERE user_id = $1 AND ($2::timestamptz IS NULL OR updated_at > $2::timestamptz)
ORDER BY updated_at`

// GetByUser returns one chunk of the user's encrypted notes.
func (c NoteRepo) GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Note, error) {
	var cursor any
	if lastID != "" {
		cursor = lastID
	}

	rows, err := c.q(ctx).QueryContext(ctx, NoteListByUserQuery, user.ID, cursor, limit, offset)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Note.GetByUser.Query", err)
	}
	defer rows.Close()

	var notes []*model.Note
	for rows.Next() {
		var note model.Note
		if err := rows.Scan(
			&note.ID, &note.UserID, &note.Data, &note.Version, &note.CreatedAt, &note.UpdatedAt,
		); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Note.GetByUser.Scan", err)
		}
		notes = append(notes, &note)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Note.GetByUser.Rows", err)
	}
	return notes, nil
}

// GetByID returns a single encrypted note owned by the user.
func (c NoteRepo) GetByID(ctx context.Context, user *model.User, id string) (*model.Note, error) {
	var note model.Note
	err := c.q(ctx).QueryRowContext(ctx, NoteGetByIDQuery, id, user.ID).
		Scan(&note.ID, &note.UserID, &note.Data, &note.Version, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoteNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Note.GetByID", err)
	}
	return &note, nil
}

// Create stores a new encrypted note for the user.
func (c NoteRepo) Create(ctx context.Context, user *model.User, note *model.Note) (*model.Note, error) {
	err := c.q(ctx).QueryRowContext(ctx, NoteCreateQuery, note.ID, user.ID, note.Data).
		Scan(&note.ID, &note.UserID, &note.Data, &note.Version, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoteNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Note.Create", err)
	}
	return note, nil
}

// Update overwrites the note's encrypted payload using optimistic versioning.
func (c NoteRepo) Update(ctx context.Context, user *model.User, note *model.Note) (*model.Note, error) {
	err := c.q(ctx).QueryRowContext(ctx, NoteUpdateQuery, note.Data, note.ID, user.ID, note.Version).
		Scan(&note.ID, &note.UserID, &note.Data, &note.Version, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVersionConflict
		}
		return nil, labelerrors.NewLabelError(labelRepository+".Note.Update", err)
	}
	return note, nil
}

// Delete removes a note owned by the user.
func (c NoteRepo) Delete(ctx context.Context, user *model.User, id string) error {
	res, err := c.q(ctx).ExecContext(ctx, NoteDeleteQuery, id, user.ID)
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Note.Delete", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".Note.Delete.RowsAffected", err)
	}
	if affected == 0 {
		return model.ErrNoteNotFound
	}
	return nil
}

// Changes returns all the user's notes (including tombstones) updated after since.
func (c NoteRepo) Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.NoteChange, error) {
	var cursor any
	if !since.IsZero() {
		cursor = since
	}

	rows, err := c.q(ctx).QueryContext(ctx, NoteChangesQuery, user.ID, cursor)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Note.Changes.Query", err)
	}
	defer rows.Close()

	var changes []*model.NoteChange
	for rows.Next() {
		var ch model.NoteChange
		if err := rows.Scan(&ch.ID, &ch.Data, &ch.Version, &ch.Deleted, &ch.UpdatedAt); err != nil {
			return nil, labelerrors.NewLabelError(labelRepository+".Note.Changes.Scan", err)
		}
		changes = append(changes, &ch)
	}
	if err := rows.Err(); err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".Note.Changes.Rows", err)
	}
	return changes, nil
}
