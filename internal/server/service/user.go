package service

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/model"
)

type UserRepository interface {
	Create(ctx context.Context, u model.User) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (model.User, error)
}

// Registrar persists a user together with its initial refresh token in one
// transaction and returns the plaintext refresh token.
type Registrar interface {
	Register(ctx context.Context, u model.User) (*model.User, string, error)
}

// TokenIssuer signs short-lived access tokens for a user.
type TokenIssuer interface {
	Issue(userID string) (string, error)
}

// Tokens is the pair of credentials handed to the client after auth.
type Tokens struct {
	Access  string
	Refresh string
}

type UserService struct {
	Repo      UserRepository
	Registrar Registrar
	Issuer    TokenIssuer
}

func NewUserService(repo UserRepository, registrar Registrar, issuer TokenIssuer) *UserService {
	return &UserService{Repo: repo, Registrar: registrar, Issuer: issuer}
}

// Register creates the user with its first refresh token (atomically) and then
// issues a signed access token for the new account.
func (s *UserService) Register(ctx context.Context, u model.User) (*model.User, Tokens, error) {
	user, refresh, err := s.Registrar.Register(ctx, u)
	if err != nil {
		return nil, Tokens{}, err
	}
	access, err := s.Issuer.Issue(user.ID)
	if err != nil {
		return nil, Tokens{}, fmt.Errorf("issue access token: %w", err)
	}
	return user, Tokens{Access: access, Refresh: refresh}, nil
}
