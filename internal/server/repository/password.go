package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
)

type PassRepo struct {
	repoBase
}

func (p PassRepo) GetByUser(ctx context.Context, user *model.User, lastID string, limit, offset int) ([]*model.Password, error) {
	//TODO implement me
	panic("implement me")
}

func (p PassRepo) GetByID(ctx context.Context, user *model.User, id string) (*model.Password, error) {
	//TODO implement me
	panic("implement me")
}

func (p PassRepo) Create(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error) {
	//TODO implement me
	panic("implement me")
}

func (p PassRepo) Update(ctx context.Context, user *model.User, pass *model.Password) (*model.Password, error) {
	//TODO implement me
	panic("implement me")
}

func (p PassRepo) Delete(ctx context.Context, user *model.User, id string) error {
	//TODO implement me
	panic("implement me")
}

func NewPasswordRepo(db *conn.DB) *PassRepo {
	return &PassRepo{repoBase: repoBase{db: db}}
}
