package grpc

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/password/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newPasswordServer(repo entryStoreStub) *PasswordServer {
	return NewPasswordServer(PasswordServerProp{
		Service: service.NewEntryService(repo),
		Logger:  slog.New(slog.DiscardHandler),
	})
}

func TestPasswordServerAdd(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.Add(context.Background(), &pb.AddRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			create: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Add(authed("u1"), &pb.AddRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			create: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return &model.Entry{ID: "p1", Version: 1}, nil
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
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.Get(context.Background(), &pb.GetRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return nil, model.ErrEntryNotFound
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return &model.Entry{ID: "p1"}, nil
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
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.List(context.Background(), &pb.ListRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			getByUser: func(_ context.Context, _ string, _ string, _, _ int) ([]*model.Entry, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.List(authed("u1"), &pb.ListRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			getByUser: func(_ context.Context, _ string, _ string, _, _ int) ([]*model.Entry, error) {
				return []*model.Entry{{ID: "p1"}, {ID: "p2"}}, nil
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
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.Update(context.Background(), &pb.UpdateRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("version conflict", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			update: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return nil, model.ErrVersionConflict
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Aborted {
			t.Fatalf("code = %v, want Aborted", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			update: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			update: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return &model.Entry{ID: "p1", Version: 2}, nil
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
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.Delete(context.Background(), &pb.DeleteRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return model.ErrEntryNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return errors.New("db down") },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestPasswordServerChanges(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{})
		_, err := srv.Changes(context.Background(), &pb.ChangesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			changes: func(_ context.Context, _ string, _ time.Time) ([]*model.EntryChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newPasswordServer(entryStoreStub{
			changes: func(_ context.Context, _ string, _ time.Time) ([]*model.EntryChange, error) {
				return []*model.EntryChange{{ID: "p1"}}, nil
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
