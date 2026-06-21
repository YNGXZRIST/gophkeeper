package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/password/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type passwordRepoStub struct {
	getByUser func(context.Context, *model.User, string, int, int) ([]*model.Password, error)
	getByID   func(context.Context, *model.User, string) (*model.Password, error)
	create    func(context.Context, *model.User, *model.Password) (*model.Password, error)
	update    func(context.Context, *model.User, *model.Password) (*model.Password, error)
	del       func(context.Context, *model.User, string) error
	changes   func(context.Context, *model.User, time.Time) ([]*model.PasswordChange, error)
}

func (s passwordRepoStub) GetByUser(ctx context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Password, error) {
	return s.getByUser(ctx, u, lastID, limit, offset)
}
func (s passwordRepoStub) GetByID(ctx context.Context, u *model.User, id string) (*model.Password, error) {
	return s.getByID(ctx, u, id)
}
func (s passwordRepoStub) Create(ctx context.Context, u *model.User, p *model.Password) (*model.Password, error) {
	return s.create(ctx, u, p)
}
func (s passwordRepoStub) Update(ctx context.Context, u *model.User, p *model.Password) (*model.Password, error) {
	return s.update(ctx, u, p)
}
func (s passwordRepoStub) Delete(ctx context.Context, u *model.User, id string) error {
	return s.del(ctx, u, id)
}
func (s passwordRepoStub) Changes(ctx context.Context, u *model.User, since time.Time) ([]*model.PasswordChange, error) {
	if s.changes == nil {
		return nil, nil
	}
	return s.changes(ctx, u, since)
}

func newPasswordServer(repo passwordRepoStub) *PasswordServer {
	return NewPasswordServer(PasswordServerProp{
		Service: service.NewPasswordService(repo),
		Logger:  zap.NewNop(),
	})
}

func TestPasswordServerAdd(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.Add(context.Background(), &pb.AddRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			create: func(_ context.Context, _ *model.User, _ *model.Password) (*model.Password, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Add(authed("u1"), &pb.AddRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			create: func(_ context.Context, _ *model.User, _ *model.Password) (*model.Password, error) {
				return &model.Password{ID: "p1", Version: 1}, nil
			},
		})
		resp, err := srv.Add(authed("u1"), &pb.AddRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetPassword().GetId() != "p1" {
			t.Fatalf("id = %q, want p1", resp.GetPassword().GetId())
		}
	})
}

func TestPasswordServerGet(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.Get(context.Background(), &pb.GetRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Password, error) {
				return nil, model.ErrPasswordNotFound
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Password, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Password, error) {
				return &model.Password{ID: "p1"}, nil
			},
		})
		resp, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetPassword().GetId() != "p1" {
			t.Fatalf("id = %q, want p1", resp.GetPassword().GetId())
		}
	})
}

func TestPasswordServerList(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.List(context.Background(), &pb.ListRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.Password, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.List(authed("u1"), &pb.ListRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.Password, error) {
				return []*model.Password{{ID: "p1"}, {ID: "p2"}}, nil
			},
		})
		resp, err := srv.List(authed("u1"), &pb.ListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.GetPasswords()) != 2 {
			t.Fatalf("passwords = %d, want 2", len(resp.GetPasswords()))
		}
	})
}

func TestPasswordServerUpdate(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.Update(context.Background(), &pb.UpdateRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("version conflict", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Password) (*model.Password, error) {
				return nil, model.ErrVersionConflict
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Aborted {
			t.Fatalf("code = %v, want Aborted", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Password) (*model.Password, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Password) (*model.Password, error) {
				return &model.Password{ID: "p1", Version: 2}, nil
			},
		})
		resp, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetPassword().GetVersion() != 2 {
			t.Fatalf("version = %d, want 2", resp.GetPassword().GetVersion())
		}
	})
}

func TestPasswordServerDelete(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.Delete(context.Background(), &pb.DeleteRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return model.ErrPasswordNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return errors.New("db down") },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestPasswordServerChanges(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{})
		_, err := srv.Changes(context.Background(), &pb.ChangesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.PasswordChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(passwordRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.PasswordChange, error) {
				return []*model.PasswordChange{{ID: "p1"}}, nil
			},
		})
		resp, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.GetChanges()) != 1 {
			t.Fatalf("changes = %d, want 1", len(resp.GetChanges()))
		}
	})
}
