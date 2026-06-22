package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/server/auth/hasher"
	"gophkeeper/internal/server/model"
)

type userRepository interface {
	Create(ctx context.Context, u model.User) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

// authWriter persists refresh tokens: Register creates a new user with its first
// token (atomically), IssueRefresh adds a token for an existing user on login.
// Both return the plaintext refresh token.
type authWriter interface {
	Register(ctx context.Context, u model.User) (*model.User, string, error)
	IssueRefresh(ctx context.Context, userID string) (string, error)
	Rotate(ctx context.Context, refreshToken string) (string, string, error)
}

// tokenIssuer signs short-lived access tokens for a user.
type tokenIssuer interface {
	Issue(userID string) (string, error)
}

// Tokens is the pair of credentials handed to the client after auth.
type Tokens struct {
	Access  string
	Refresh string
}

type UserService struct {
	repo   userRepository
	auth   authWriter
	issuer tokenIssuer
}

func NewUserService(repo userRepository, auth authWriter, issuer tokenIssuer) *UserService {
	return &UserService{repo: repo, auth: auth, issuer: issuer}
}

// Register creates the user with its first refresh token (atomically) and then
// issues a signed access token for the new account.
func (s *UserService) Register(ctx context.Context, u model.User) (*model.User, Tokens, error) {
	user, refresh, err := s.auth.Register(ctx, u)
	if err != nil {
		return nil, Tokens{}, err
	}
	access, err := s.issuer.Issue(user.ID)
	if err != nil {
		return nil, Tokens{}, fmt.Errorf("issue access token: %w", err)
	}
	return user, Tokens{Access: access, Refresh: refresh}, nil
}

// Login verifies credentials and issues a fresh token pair, adding a new refresh
// token for this device without affecting the user's other sessions. A missing
// user or wrong password both yield model.ErrInvalidCredentials, so the caller
// cannot tell them apart.
func (s *UserService) Login(ctx context.Context, login, password string) (*model.User, Tokens, error) {
	user, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, Tokens{}, model.ErrInvalidCredentials
		}
		return nil, Tokens{}, fmt.Errorf("get user by login: %w", err)
	}
	if err = hasher.Compare(user.Pass, password); err != nil {
		return nil, Tokens{}, model.ErrInvalidCredentials
	}
	refresh, err := s.auth.IssueRefresh(ctx, user.ID)
	if err != nil {
		return nil, Tokens{}, fmt.Errorf("issue refresh token: %w", err)
	}
	access, err := s.issuer.Issue(user.ID)
	if err != nil {
		return nil, Tokens{}, fmt.Errorf("issue access token: %w", err)
	}
	return user, Tokens{Access: access, Refresh: refresh}, nil
}

// Refresh rotates a refresh token and issues a matching new access token. An
// unknown or expired refresh token yields model.ErrInvalidRefreshToken.
func (s *UserService) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	userID, refresh, err := s.auth.Rotate(ctx, refreshToken)
	if err != nil {
		return Tokens{}, err
	}
	access, err := s.issuer.Issue(userID)
	if err != nil {
		return Tokens{}, fmt.Errorf("issue access token: %w", err)
	}
	return Tokens{Access: access, Refresh: refresh}, nil
}
