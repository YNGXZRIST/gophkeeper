package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gophkeeper/internal/server/auth/hasher"
	"gophkeeper/internal/server/model"
)

type repoStub struct {
	create     func(context.Context, model.User) (*model.User, error)
	getByLogin func(context.Context, string) (*model.User, error)
}

func (s repoStub) Create(ctx context.Context, u model.User) (*model.User, error) {
	return s.create(ctx, u)
}

func (s repoStub) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	return s.getByLogin(ctx, login)
}

type authStub struct {
	register     func(context.Context, model.User) (*model.User, string, error)
	issueRefresh func(context.Context, string) (string, error)
}

func (s authStub) Register(ctx context.Context, u model.User) (*model.User, string, error) {
	return s.register(ctx, u)
}

func (s authStub) IssueRefresh(ctx context.Context, userID string) (string, error) {
	return s.issueRefresh(ctx, userID)
}

type issuerStub struct {
	issue func(string) (string, error)
}

func (s issuerStub) Issue(userID string) (string, error) {
	return s.issue(userID)
}

func TestUserServiceLogin(t *testing.T) {
	hash, err := hasher.Hash("correct")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user := &model.User{ID: "u1", Login: "alice", Pass: hash}

	tests := []struct {
		name         string
		password     string
		getByLogin   func(context.Context, string) (*model.User, error)
		issueRefresh func(context.Context, string) (string, error)
		issue        func(string) (string, error)
		want         Tokens
		expectErr    bool
		sentinel     error
	}{
		{
			name:         "success",
			password:     "correct",
			getByLogin:   func(context.Context, string) (*model.User, error) { return user, nil },
			issueRefresh: func(context.Context, string) (string, error) { return "refresh-tok", nil },
			issue:        func(string) (string, error) { return "access-tok", nil },
			want:         Tokens{Access: "access-tok", Refresh: "refresh-tok"},
		},
		{
			name:       "user not found",
			password:   "correct",
			getByLogin: func(context.Context, string) (*model.User, error) { return nil, sql.ErrNoRows },
			expectErr:  true,
			sentinel:   model.ErrInvalidCredentials,
		},
		{
			name:       "wrong password",
			password:   "wrong",
			getByLogin: func(context.Context, string) (*model.User, error) { return user, nil },
			expectErr:  true,
			sentinel:   model.ErrInvalidCredentials,
		},
		{
			name:         "refresh error",
			password:     "correct",
			getByLogin:   func(context.Context, string) (*model.User, error) { return user, nil },
			issueRefresh: func(context.Context, string) (string, error) { return "", errors.New("db down") },
			expectErr:    true,
		},
		{
			name:         "access error",
			password:     "correct",
			getByLogin:   func(context.Context, string) (*model.User, error) { return user, nil },
			issueRefresh: func(context.Context, string) (string, error) { return "refresh-tok", nil },
			issue:        func(string) (string, error) { return "", errors.New("sign failed") },
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(
				repoStub{getByLogin: tt.getByLogin},
				authStub{issueRefresh: tt.issueRefresh},
				issuerStub{issue: tt.issue},
			)
			gotUser, gotTokens, err := svc.Login(context.Background(), "alice", tt.password)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.sentinel != nil && !errors.Is(err, tt.sentinel) {
					t.Fatalf("err = %v, want %v", err, tt.sentinel)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotUser != user {
				t.Errorf("user = %v, want %v", gotUser, user)
			}
			if gotTokens != tt.want {
				t.Errorf("tokens = %v, want %v", gotTokens, tt.want)
			}
		})
	}
}

func TestUserServiceRegister(t *testing.T) {
	user := &model.User{ID: "u1", Login: "alice"}

	tests := []struct {
		name      string
		register  func(context.Context, model.User) (*model.User, string, error)
		issue     func(string) (string, error)
		want      Tokens
		expectErr bool
	}{
		{
			name:     "success",
			register: func(context.Context, model.User) (*model.User, string, error) { return user, "refresh-tok", nil },
			issue:    func(string) (string, error) { return "access-tok", nil },
			want:     Tokens{Access: "access-tok", Refresh: "refresh-tok"},
		},
		{
			name:      "register error",
			register:  func(context.Context, model.User) (*model.User, string, error) { return nil, "", errors.New("dup") },
			expectErr: true,
		},
		{
			name:      "access error",
			register:  func(context.Context, model.User) (*model.User, string, error) { return user, "refresh-tok", nil },
			issue:     func(string) (string, error) { return "", errors.New("sign failed") },
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(
				repoStub{},
				authStub{register: tt.register},
				issuerStub{issue: tt.issue},
			)
			gotUser, gotTokens, err := svc.Register(context.Background(), model.User{Login: "alice"})
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotUser != user {
				t.Errorf("user = %v, want %v", gotUser, user)
			}
			if gotTokens != tt.want {
				t.Errorf("tokens = %v, want %v", gotTokens, tt.want)
			}
		})
	}
}
