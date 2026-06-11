package repository

import (
	"context"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
)

const labelRepository = "REPOSITORY"

type repoBase struct{ db *conn.DB }

func (b *repoBase) q(ctx context.Context) trmanager.Querier {
	return trmanager.Resolve(ctx, b.db)
}

type Repositories struct {
	User         *UserRepo
	RefreshToken *RefreshTokenRepo
}
