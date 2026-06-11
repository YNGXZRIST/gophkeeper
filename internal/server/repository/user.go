package repository

import (
	"context"
	"errors"
	"gophkeeper/internal/server/auth/hasher"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/shared/errors/labelerrors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepo struct {
	repoBase
}

const UserRegisterQuery = `INSERT INTO users(login, password) VALUES ($1, $2) RETURNING id, login, password, created_at, updated_at`

const UserGetByLoginQuery = `SELECT id, login, password, created_at, updated_at FROM users WHERE login = $1`

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
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, model.ErrLoginTaken
		}
		return nil, labelerrors.NewLabelError(labelRepository+".User.Register.Insert", err)
	}
	return &u, nil
}

func (ur *UserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	var u model.User
	err := ur.repoBase.q(ctx).QueryRowContext(ctx, UserGetByLoginQuery, login).
		Scan(&u.ID, &u.Login, &u.Pass, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelRepository+".User.GetByLogin", err)
	}
	return &u, nil
}
