package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
)

type UserRepo struct {
	repoBase
}

func NewUserRepo(db *conn.DB) *UserRepo {
	return &UserRepo{repoBase: repoBase{db: db}}
}
func (ur *UserRepo) Create(ctx context.Context, u model.User) error {
	return nil
}

func (ur *UserRepo) GetByLogin(ctx context.Context, login string) (model.User, error) {
	return model.User{}, nil
}
