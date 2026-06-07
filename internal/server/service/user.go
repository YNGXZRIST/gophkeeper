package service

import (
	"context"
	"gophkeeper/internal/server/model"
)

type UserRepository interface {
	Create(ctx context.Context, u model.User) error
	GetByLogin(ctx context.Context, login string) (model.User, error)
}
type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}
