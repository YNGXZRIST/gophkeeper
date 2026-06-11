package repository

import (
	"context"
	"gophkeeper/internal/server/auth/hasher"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"
)

type UserRepo struct {
	repoBase
}

const UserRegisterQuery = `INSERT INTO users(login, password) VALUES ($1, $2) RETURNING id, login, password, created_at, updated_at`

func NewUserRepo(db *conn.DB) *UserRepo {
	return &UserRepo{repoBase: repoBase{db: db}}
}
func (ur *UserRepo) Create(ctx context.Context, u model.User) (*model.User, error) {
	hash, err := hasher.Hash(u.Pass)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".User.Register.HashPassword", err)
	}
	err = ur.repoBase.q(ctx).QueryRowContext(ctx,
		UserRegisterQuery,
		u.Login, hash,
	).Scan(&u.ID, &u.Login, &u.Pass, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".User.Register.Insert", err)
	}
	return &u, nil
}

func (ur *UserRepo) GetByLogin(ctx context.Context, login string) (model.User, error) {
	return model.User{}, nil
}
