package service

import (
	"context"
	"gophkeeper/internal/server/model"
)

type noteRepository interface {
	GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Note, error)
	GetByID(ctx context.Context, user *model.User, id string) (*model.Note, error)
	Create(ctx context.Context, user *model.User, note *model.Note) (*model.Note, error)
	Update(ctx context.Context, user *model.User, note *model.Note) (*model.Note, error)
	Delete(ctx context.Context, user *model.User, id string) error
}
type NoteService struct {
	repo noteRepository
}

func NewNoteService(repo noteRepository) *NoteService {
	return &NoteService{repo: repo}
}

// Add stores a new note for the user.
func (s *NoteService) Add(ctx context.Context, userID string, data []byte) (*model.Note, error) {
	return s.repo.Create(ctx, &model.User{ID: userID}, &model.Note{Data: data})
}

// List returns a chunk of the user's notes.
func (s *NoteService) List(ctx context.Context, userID, lastID string, limit, offset int) ([]*model.Note, error) {
	return s.repo.GetByUser(ctx, &model.User{ID: userID}, lastID, limit, offset)
}

// Get returns a single note owned by the user.
func (s *NoteService) Get(ctx context.Context, userID, id string) (*model.Note, error) {
	return s.repo.GetByID(ctx, &model.User{ID: userID}, id)
}

// Update overwrites a note owned by the user.
func (s *NoteService) Update(ctx context.Context, userID, id string, data []byte, version int64) (*model.Note, error) {
	return s.repo.Update(ctx, &model.User{ID: userID}, &model.Note{ID: id, Data: data, Version: version})
}

// Delete removes a note owned by the user.
func (s *NoteService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, &model.User{ID: userID}, id)
}
