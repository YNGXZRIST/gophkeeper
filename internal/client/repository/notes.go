package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type NotesRepo struct {
	repoBase
}

func NewNotesRepo(conn *sql.DB) *NotesRepo {
	return &NotesRepo{repoBase: repoBase{db: conn}}
}

type NoteRow struct {
	ID      string
	Data    []byte
	Version int64
}

const notesListQuery = `SELECT id, data, version FROM notes
WHERE deleted = 0 AND (? = '' OR id > ?)
ORDER BY id LIMIT ?`
const notesInsertQuery = `INSERT INTO notes (id, data, version, dirty, base_version) VALUES (?, ?, 1, 1, 0)`
const notesUpdateQuery = `UPDATE notes SET data = ?, version = version + 1, dirty = 1,
base_version = CASE WHEN conflict = 1 THEN COALESCE(server_version, base_version) ELSE base_version END,
conflict = 0, server_blob = NULL, server_version = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
const notesDeleteQuery = `UPDATE notes SET deleted = 1, dirty = 1, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

func (r *NotesRepo) List(ctx context.Context, lastID string, limit int) ([]NoteRow, error) {
	rows, err := r.db.QueryContext(ctx, notesListQuery, lastID, lastID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NoteRow
	for rows.Next() {
		var n NoteRow
		if err := rows.Scan(&n.ID, &n.Data, &n.Version); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *NotesRepo) Create(ctx context.Context, data []byte) (NoteRow, error) {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		notesInsertQuery,
		id, data)
	if err != nil {
		return NoteRow{}, err
	}
	return NoteRow{ID: id, Data: data, Version: 1}, nil
}

func (r *NotesRepo) Update(ctx context.Context, id string, data []byte) error {
	_, err := r.db.ExecContext(ctx,
		notesUpdateQuery,
		data, id)
	return err
}

func (r *NotesRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		notesDeleteQuery,
		id)
	return err
}
