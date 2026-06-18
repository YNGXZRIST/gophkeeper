package service

import (
	"context"
	"gophkeeper/internal/server/model"
)

type passwordRepository interface {
	GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Password, error)
	GetByID(ctx context.Context, user *model.User, id string) (*model.Password, error)
	Create(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error)
	Update(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error)
	Delete(ctx context.Context, user *model.User, id string) error
}
type PasswordService struct {
	Repo passwordRepository
}

func NewPasswordService(repo passwordRepository) *PasswordService {
	return &PasswordService{Repo: repo}
}
