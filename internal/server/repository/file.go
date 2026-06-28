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

type FileRepo struct {
	repoBase
}

func NewFileRepo(db *conn.DB) *FileRepo {
	return &FileRepo{repoBase: repoBase{db: db}}
}

const FileListByUserQuery = `SELECT id, user_id, meta, chunk_count, version, created_at, updated_at
FROM files
WHERE user_id = $1 AND deleted_at IS NULL AND ($2::uuid IS NULL OR id > $2::uuid)
ORDER BY id
LIMIT $3 OFFSET $4`

const FileGetMetaQuery = `SELECT id, user_id, meta, chunk_count, version, created_at, updated_at
FROM files
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const FileCreateQuery = `INSERT INTO files(user_id, meta, chunk_count) VALUES ($1, $2, $3)
RETURNING id, user_id, meta, chunk_count, version, created_at, updated_at`

const FileChunkInsertQuery = `INSERT INTO file_chunks(file_id, idx, data) VALUES ($1, $2, $3)`

const FileChunksByFileQuery = `SELECT idx, data FROM file_chunks WHERE file_id = $1 ORDER BY idx`

const FileDeleteQuery = `UPDATE files
SET deleted_at = now(), version = version + 1, updated_at = now()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

const FileUpdateMetaQuery = `UPDATE files
SET meta = $1, version = version + 1, updated_at = now()
WHERE id = $2 AND user_id = $3 AND version = $4
RETURNING id, user_id, meta, chunk_count, version, created_at, updated_at`

const FileChangesQuery = `SELECT id, meta, version, deleted_at IS NOT NULL, updated_at
FROM files
WHERE user_id = $1 AND ($2::timestamptz IS NULL OR updated_at > $2::timestamptz)
ORDER BY updated_at`

// UpdateMeta overwrites a file's encrypted metadata using optimistic versioning.
func (r FileRepo) UpdateMeta(ctx context.Context, user *model.User, id string, meta []byte, version int64) (*model.File, error) {
	var file model.File
	err := r.q(ctx).QueryRowContext(ctx, FileUpdateMetaQuery, meta, id, user.ID, version).
		Scan(&file.ID, &file.UserID, &file.Meta, &file.ChunkCount, &file.Version, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrVersionConflict
		}
		return nil, labelerrors.NewLabelError(labelRepository+".File.UpdateMeta", err)
	}
	return &file, nil
}

// CreateFile inserts the file metadata row; chunks are inserted separately
// within the same transaction.
func (r FileRepo) CreateFile(ctx context.Context, user *model.User, meta []byte, chunkCount int) (*model.File, error) {
	var file model.File
	err := r.q(ctx).QueryRowContext(ctx, FileCreateQuery, user.ID, meta, chunkCount).
		Scan(&file.ID, &file.UserID, &file.Meta, &file.ChunkCount, &file.Version, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".File.CreateFile", err)
	}
	return &file, nil
}

// InsertChunk stores one encrypted chunk of a file.
func (r FileRepo) InsertChunk(ctx context.Context, fileID string, idx int, data []byte) error {
	if _, err := r.q(ctx).ExecContext(ctx, FileChunkInsertQuery, fileID, idx, data); err != nil {
		return labelerrors.NewLabelError(labelRepository+".File.InsertChunk", err)
	}
	return nil
}

// GetByUser returns one chunk of the user's file metadata, without chunk bodies.
func (r FileRepo) GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.File, error) {
	var cursor any
	if lastID != "" {
		cursor = lastID
	}

	return queryRows(ctx, r.q(ctx), labelRepository+".File.GetByUser", FileListByUserQuery, scanFile, user.ID, cursor, limit, offset)
}

func scanFile(s scanner) (*model.File, error) {
	var file model.File
	if err := s.Scan(&file.ID, &file.UserID, &file.Meta, &file.ChunkCount, &file.Version, &file.CreatedAt, &file.UpdatedAt); err != nil {
		return nil, err
	}
	return &file, nil
}

// GetMeta returns a single file's metadata owned by the user, without chunk bodies.
func (r FileRepo) GetMeta(ctx context.Context, user *model.User, id string) (*model.File, error) {
	var file model.File
	err := r.q(ctx).QueryRowContext(ctx, FileGetMetaQuery, id, user.ID).
		Scan(&file.ID, &file.UserID, &file.Meta, &file.ChunkCount, &file.Version, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrFileNotFound
		}
		return nil, labelerrors.NewLabelError(labelRepository+".File.GetMeta", err)
	}
	return &file, nil
}

// StreamChunks reads the file's chunks in order and passes each to fn, never
// holding more than one chunk in memory.
func (r FileRepo) StreamChunks(ctx context.Context, fileID string, fn func(idx int, data []byte) error) error {
	rows, err := r.q(ctx).QueryContext(ctx, FileChunksByFileQuery, fileID)
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".File.StreamChunks.Query", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			idx  int
			data []byte
		)
		if err := rows.Scan(&idx, &data); err != nil {
			return labelerrors.NewLabelError(labelRepository+".File.StreamChunks.Scan", err)
		}
		if err := fn(idx, data); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return labelerrors.NewLabelError(labelRepository+".File.StreamChunks.Rows", err)
	}
	return nil
}

// Delete soft-deletes a file owned by the user by marking it as a tombstone.
func (r FileRepo) Delete(ctx context.Context, user *model.User, id string) error {
	res, err := r.q(ctx).ExecContext(ctx, FileDeleteQuery, id, user.ID)
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".File.Delete", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return labelerrors.NewLabelError(labelRepository+".File.Delete.RowsAffected", err)
	}
	if affected == 0 {
		return model.ErrFileNotFound
	}
	return nil
}

// Changes returns all the user's file metadata (including tombstones) updated after since.
func (r FileRepo) Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.FileChange, error) {
	var cursor any
	if !since.IsZero() {
		cursor = since
	}

	return queryRows(ctx, r.q(ctx), labelRepository+".File.Changes", FileChangesQuery, scanFileChange, user.ID, cursor)
}

func scanFileChange(s scanner) (*model.FileChange, error) {
	var ch model.FileChange
	if err := s.Scan(&ch.ID, &ch.Meta, &ch.Version, &ch.Deleted, &ch.UpdatedAt); err != nil {
		return nil, err
	}
	return &ch, nil
}
