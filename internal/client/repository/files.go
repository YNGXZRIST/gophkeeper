package repository

import (
	"context"
	"database/sql"
)

type FilesRepo struct {
	syncRepo
}

func NewFilesRepo(conn *sql.DB) *FilesRepo {
	return &FilesRepo{syncRepo: newSyncRepo(conn, "files", "meta")}
}

type FileRow struct {
	ID         string
	Meta       []byte
	ChunkCount int
	Version    int64
}

const filesListQuery = `SELECT id, meta, chunk_count, version FROM files
WHERE deleted = 0 AND (? = '' OR id > ?)
ORDER BY id LIMIT ?`
const filesInsertQuery = `INSERT INTO files (id, meta, chunk_count, version, dirty, base_version) VALUES (?, ?, ?, ?, 0, ?)
ON CONFLICT(id) DO UPDATE SET meta = excluded.meta, chunk_count = excluded.chunk_count, version = excluded.version, base_version = excluded.version, dirty = 0, deleted = 0, conflict = 0, server_blob = NULL, server_version = NULL`
const filesUpdateMetaQuery = `UPDATE files SET meta = ?, version = version + 1, dirty = 1,
base_version = CASE WHEN conflict = 1 THEN COALESCE(server_version, base_version) ELSE base_version END,
conflict = 0, server_blob = NULL, server_version = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
const filesDeleteQuery = `UPDATE files SET deleted = 1, dirty = 1, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`

func scanFileRow(s scanner) (FileRow, error) {
	var f FileRow
	err := s.Scan(&f.ID, &f.Meta, &f.ChunkCount, &f.Version)
	return f, err
}

func (r *FilesRepo) List(ctx context.Context, lastID string, limit int) ([]FileRow, error) {
	return queryRows(ctx, r.db, filesListQuery, scanFileRow, lastID, lastID, limit)
}

func (r *FilesRepo) Insert(ctx context.Context, id string, meta []byte, chunkCount int, version int64) error {
	_, err := r.db.ExecContext(ctx, filesInsertQuery, id, meta, chunkCount, version, version)
	return err
}

func (r *FilesRepo) UpdateMeta(ctx context.Context, id string, meta []byte) error {
	_, err := r.db.ExecContext(ctx, filesUpdateMetaQuery, meta, id)
	return err
}

func (r *FilesRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, filesDeleteQuery, id)
	return err
}
