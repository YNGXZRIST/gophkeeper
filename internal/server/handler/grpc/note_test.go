package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/note/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newNoteServer(repo entryStoreStub) *NoteServer {
	return NewNoteServer(NoteServerProp{
		Service: service.NewEntryService(repo),
		Logger:  zap.NewNop(),
	})
}

func TestNoteServerAdd(t *testing.T) {
	srv := newNoteServer(entryStoreStub{
		create: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
			return &model.Entry{ID: "n1", Version: 1}, nil
		},
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := srv.Add(context.Background(), &pb.AddRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		resp, err := srv.Add(authed("u1"), &pb.AddRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetNote().GetId() != "n1" {
			t.Fatalf("id = %q, want n1", resp.GetNote().GetId())
		}
	})
}

func TestNoteServerGet(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{})
		_, err := srv.Get(context.Background(), &pb.GetRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return &model.Entry{ID: "n1"}, nil
			},
		})
		resp, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetNote().GetId() != "n1" {
			t.Fatalf("id = %q, want n1", resp.GetNote().GetId())
		}
	})
}

func TestNoteServerList(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{})
		_, err := srv.List(context.Background(), &pb.ListRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
			getByUser: func(_ context.Context, _ string, _ string, _, _ int) ([]*model.Entry, error) {
				return []*model.Entry{{ID: "n1"}, {ID: "n2"}}, nil
			},
		})
		resp, err := srv.List(authed("u1"), &pb.ListRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.GetNotes()) != 2 {
			t.Fatalf("notes = %d, want 2", len(resp.GetNotes()))
		}
	})
}

func TestNoteServerUpdate(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{})
		_, err := srv.Update(context.Background(), &pb.UpdateRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("version conflict", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
			update: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
				return &model.Entry{ID: "n1", Version: 2}, nil
			},
		})
		resp, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetNote().GetVersion() != 2 {
			t.Fatalf("version = %d, want 2", resp.GetNote().GetVersion())
		}
	})
}

func TestNoteServerDelete(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{})
		_, err := srv.Delete(context.Background(), &pb.DeleteRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return model.ErrEntryNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return errors.New("db down") },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNoteServerChanges(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{})
		_, err := srv.Changes(context.Background(), &pb.ChangesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(entryStoreStub{
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
		srv := newNoteServer(entryStoreStub{
			changes: func(_ context.Context, _ string, _ time.Time) ([]*model.EntryChange, error) {
				return []*model.EntryChange{{ID: "n1"}}, nil
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
