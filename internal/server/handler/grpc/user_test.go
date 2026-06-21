package grpc

import (
	"context"
	"errors"
	"testing"

	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userRepoStub struct {
	create     func(context.Context, model.User) (*model.User, error)
	getByLogin func(context.Context, string) (*model.User, error)
}

func (s userRepoStub) Create(ctx context.Context, u model.User) (*model.User, error) {
	return s.create(ctx, u)
}
func (s userRepoStub) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	return s.getByLogin(ctx, login)
}

type userAuthStub struct {
	register     func(context.Context, model.User) (*model.User, string, error)
	issueRefresh func(context.Context, string) (string, error)
	rotate       func(context.Context, string) (string, string, error)
}

func (s userAuthStub) Register(ctx context.Context, u model.User) (*model.User, string, error) {
	return s.register(ctx, u)
}
func (s userAuthStub) IssueRefresh(ctx context.Context, userID string) (string, error) {
	return s.issueRefresh(ctx, userID)
}
func (s userAuthStub) Rotate(ctx context.Context, refreshToken string) (string, string, error) {
	return s.rotate(ctx, refreshToken)
}

type userIssuerStub struct {
	issue func(string) (string, error)
}

func (s userIssuerStub) Issue(userID string) (string, error) {
	return s.issue(userID)
}

func newUserServer(repo userRepoStub, auth userAuthStub, issuer userIssuerStub) *UserServer {
	return NewUserServer(UserServerProp{
		Service: service.NewUserService(repo, auth, issuer),
		Logger:  zap.NewNop(),
	})
}

func TestUserServerRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{register: func(context.Context, model.User) (*model.User, string, error) {
				return &model.User{ID: "u1", Login: "alice"}, "refresh-tok", nil
			}},
			userIssuerStub{issue: func(string) (string, error) { return "access-tok", nil }},
		)
		resp, err := srv.Register(context.Background(), &pb.RegisterRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetUser().GetId() != "u1" || resp.GetAccessToken() != "access-tok" || resp.GetRefreshToken() != "refresh-tok" {
			t.Fatalf("resp = %+v, want id=u1 access=access-tok refresh=refresh-tok", resp)
		}
	})

	t.Run("login taken", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{register: func(context.Context, model.User) (*model.User, string, error) {
				return nil, "", model.ErrLoginTaken
			}},
			userIssuerStub{},
		)
		_, err := srv.Register(context.Background(), &pb.RegisterRequest{})
		if status.Code(err) != codes.AlreadyExists {
			t.Fatalf("code = %v, want AlreadyExists", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{register: func(context.Context, model.User) (*model.User, string, error) {
				return nil, "", errors.New("db down")
			}},
			userIssuerStub{},
		)
		_, err := srv.Register(context.Background(), &pb.RegisterRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})
}

func TestUserServerLogin(t *testing.T) {
	t.Run("invalid credentials", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{getByLogin: func(context.Context, string) (*model.User, error) {
				return nil, model.ErrInvalidCredentials
			}},
			userAuthStub{},
			userIssuerStub{},
		)
		_, err := srv.Login(context.Background(), &pb.LoginRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{getByLogin: func(context.Context, string) (*model.User, error) {
				return nil, errors.New("db down")
			}},
			userAuthStub{},
			userIssuerStub{},
		)
		_, err := srv.Login(context.Background(), &pb.LoginRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})
}

func TestUserServerRefresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{rotate: func(context.Context, string) (string, string, error) {
				return "u1", "new-refresh", nil
			}},
			userIssuerStub{issue: func(string) (string, error) { return "new-access", nil }},
		)
		resp, err := srv.Refresh(context.Background(), &pb.RefreshRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetAccessToken() != "new-access" || resp.GetRefreshToken() != "new-refresh" {
			t.Fatalf("resp = %+v, want access=new-access refresh=new-refresh", resp)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{rotate: func(context.Context, string) (string, string, error) {
				return "", "", model.ErrInvalidRefreshToken
			}},
			userIssuerStub{},
		)
		_, err := srv.Refresh(context.Background(), &pb.RefreshRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newUserServer(
			userRepoStub{},
			userAuthStub{rotate: func(context.Context, string) (string, string, error) {
				return "", "", errors.New("db down")
			}},
			userIssuerStub{},
		)
		_, err := srv.Refresh(context.Background(), &pb.RefreshRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})
}
