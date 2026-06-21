package service

import (
	"context"
	"gophkeeper/internal/server/model"
	"time"
)

type passwordRepository interface {
	GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Password, error)
	GetByID(ctx context.Context, user *model.User, id string) (*model.Password, error)
	Create(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error)
	Update(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error)
	Delete(ctx context.Context, user *model.User, id string) error
	Changes(ctx context.Context, user *model.User, since time.Time) ([]*model.PasswordChange, error)
}

type PasswordService struct {
	repo passwordRepository
}

func NewPasswordService(repo passwordRepository) *PasswordService {
	return &PasswordService{repo: repo}
}

// Add stores a new password for the user.
func (s *PasswordService) Add(ctx context.Context, userID, id string, data []byte) (*model.Password, error) {
	return s.repo.Create(ctx, &model.User{ID: userID}, &model.Password{ID: id, Data: data})
}

// List returns a chunk of the user's passwords.
func (s *PasswordService) List(ctx context.Context, userID, lastID string, limit, offset int) ([]*model.Password, error) {
	return s.repo.GetByUser(ctx, &model.User{ID: userID}, lastID, limit, offset)
}

// Get returns a single password owned by the user.
func (s *PasswordService) Get(ctx context.Context, userID, id string) (*model.Password, error) {
	return s.repo.GetByID(ctx, &model.User{ID: userID}, id)
}

// Update overwrites a password owned by the user.
func (s *PasswordService) Update(ctx context.Context, userID, id string, data []byte, version int64) (*model.Password, error) {
	return s.repo.Update(ctx, &model.User{ID: userID}, &model.Password{ID: id, Data: data, Version: version})
}

// Delete removes a password owned by the user.
func (s *PasswordService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, &model.User{ID: userID}, id)
}

// Changes returns the user's passwords changed since the given cursor (RFC3339, empty = all).
func (s *PasswordService) Changes(ctx context.Context, userID, since string) ([]*model.PasswordChange, error) {
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
