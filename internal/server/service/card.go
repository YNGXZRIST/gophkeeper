package service

import (
	"context"
	"gophkeeper/internal/server/model"
)

type CardRepository interface {
	GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Card, error)
	GetByID(ctx context.Context, user *model.User, id string) (*model.Card, error)
	Create(ctx context.Context, user *model.User, card *model.Card) (*model.Card, error)
	Update(ctx context.Context, user *model.User, card *model.Card) (*model.Card, error)
	Delete(ctx context.Context, user *model.User, id string) error
}
type CardService struct {
	Repo CardRepository
}

func NewCardService(repo CardRepository) *CardService {
	return &CardService{Repo: repo}
}

// Add stores a new card for the user.
func (s *CardService) Add(ctx context.Context, userID string, data []byte) (*model.Card, error) {
	return s.Repo.Create(ctx, &model.User{ID: userID}, &model.Card{Data: data})
}

// List returns a chunk of the user's cards.
func (s *CardService) List(ctx context.Context, userID, lastID string, limit, offset int) ([]*model.Card, error) {
	return s.Repo.GetByUser(ctx, &model.User{ID: userID}, lastID, limit, offset)
}

// Get returns a single card owned by the user.
func (s *CardService) Get(ctx context.Context, userID, id string) (*model.Card, error) {
	return s.Repo.GetByID(ctx, &model.User{ID: userID}, id)
}

// Update overwrites a card owned by the user.
func (s *CardService) Update(ctx context.Context, userID, id string, data []byte, version int64) (*model.Card, error) {
	return s.Repo.Update(ctx, &model.User{ID: userID}, &model.Card{ID: id, Data: data, Version: version})
}

// Delete removes a card owned by the user.
func (s *CardService) Delete(ctx context.Context, userID, id string) error {
	return s.Repo.Delete(ctx, &model.User{ID: userID}, id)
}
