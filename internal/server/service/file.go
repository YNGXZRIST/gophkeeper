package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/server/model"
	"io"
	"time"
)

type fileRepository interface {
	CreateFile(ctx context.Context, user *model.User, meta []byte, chunkCount int) (*model.File, error)
	InsertChunk(ctx context.Context, fileID string, idx int, data []byte) error
	GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.File, error)
	GetMeta(ctx context.Context, user *model.User, id string) (*model.File, error)
	StreamChunks(ctx context.Context, fileID string, fn func(idx int, data []byte) error) error
	UpdateMeta(ctx context.Context, user *model.User, id string, meta []byte, version int64) (*model.File, error)
	Delete(ctx context.Context, user *model.User, id string) error
	Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.FileChange, error)
}

type txRunner interface {
	WithinTx(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error
}

type FileService struct {
	repo fileRepository
	tx   txRunner
}

func NewFileService(repo fileRepository, tx txRunner) *FileService {
	return &FileService{repo: repo, tx: tx}
}

// Create stores a file atomically: the metadata row and every chunk pulled from
// next are written in one transaction, so a mid-upload failure leaves nothing.
// next returns io.EOF when no chunks remain.
func (s *FileService) Create(ctx context.Context, userID string, meta []byte, chunkCount int, next func() ([]byte, error)) (string, error) {
	var id string
	err := s.tx.WithinTx(ctx, nil, func(ctx context.Context) error {
		file, err := s.repo.CreateFile(ctx, &model.User{ID: userID}, meta, chunkCount)
		if err != nil {
			return err
		}
		id = file.ID
		for idx := 0; ; idx++ {
			data, err := next()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("get next: %w", err)
			}
			if err := s.repo.InsertChunk(ctx, file.ID, idx, data); err != nil {
				return err
			}
		}
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// List returns a chunk of the user's file metadata.
func (s *FileService) List(ctx context.Context, userID, lastID string, limit, offset int) ([]*model.File, error) {
	return s.repo.GetByUser(ctx, &model.User{ID: userID}, lastID, limit, offset)
}

// Download streams a file's metadata and then its chunks in order to the given
// callbacks. Ownership is verified before any chunk is read.
func (s *FileService) Download(ctx context.Context, userID, id string, sendMeta func([]byte) error, sendChunk func(idx int, data []byte) error) error {
	file, err := s.repo.GetMeta(ctx, &model.User{ID: userID}, id)
	if err != nil {
		return err
	}
	if err := sendMeta(file.Meta); err != nil {
		return err
	}
	return s.repo.StreamChunks(ctx, file.ID, sendChunk)
}

// UpdateMeta overwrites a file's encrypted metadata.
func (s *FileService) UpdateMeta(ctx context.Context, userID, id string, meta []byte, version int64) (*model.File, error) {
	return s.repo.UpdateMeta(ctx, &model.User{ID: userID}, id, meta, version)
}

// Delete removes a file owned by the user.
func (s *FileService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, &model.User{ID: userID}, id)
}

// Changes returns the user's file metadata changed since the given cursor (RFC3339, empty = all).
func (s *FileService) Changes(ctx context.Context, userID, since string) ([]*model.FileChange, error) {
	var t time.Time
	if since != "" {
		parsed, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return nil, err
		}
		t = parsed
	}
	return s.repo.Changes(ctx, &model.User{ID: userID}, t)
}
