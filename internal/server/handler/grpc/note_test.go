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

type noteRepoStub struct {
	getByUser func(context.Context, *model.User, string, int, int) ([]*model.Note, error)
	getByID   func(context.Context, *model.User, string) (*model.Note, error)
	create    func(context.Context, *model.User, *model.Note) (*model.Note, error)
	update    func(context.Context, *model.User, *model.Note) (*model.Note, error)
	del       func(context.Context, *model.User, string) error
	changes   func(context.Context, *model.User, time.Time) ([]*model.NoteChange, error)
}

func (s noteRepoStub) GetByUser(ctx context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Note, error) {
	return s.getByUser(ctx, u, lastID, limit, offset)
}
func (s noteRepoStub) GetByID(ctx context.Context, u *model.User, id string) (*model.Note, error) {
	return s.getByID(ctx, u, id)
}
func (s noteRepoStub) Create(ctx context.Context, u *model.User, n *model.Note) (*model.Note, error) {
	return s.create(ctx, u, n)
}
func (s noteRepoStub) Update(ctx context.Context, u *model.User, n *model.Note) (*model.Note, error) {
	return s.update(ctx, u, n)
}
func (s noteRepoStub) Delete(ctx context.Context, u *model.User, id string) error {
	return s.del(ctx, u, id)
}
func (s noteRepoStub) Changes(ctx context.Context, u *model.User, since time.Time) ([]*model.NoteChange, error) {
	if s.changes == nil {
		return nil, nil
	}
	return s.changes(ctx, u, since)
}

func newNoteServer(repo noteRepoStub) *NoteServer {
	return NewNoteServer(NoteServerProp{
		Service: service.NewNoteService(repo),
		Logger:  zap.NewNop(),
	})
}

func TestNoteServerAdd(t *testing.T) {
	srv := newNoteServer(noteRepoStub{
		create: func(_ context.Context, _ *model.User, _ *model.Note) (*model.Note, error) {
			return &model.Note{ID: "n1", Version: 1}, nil
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
		srv := newNoteServer(noteRepoStub{})
		_, err := srv.Get(context.Background(), &pb.GetRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Note, error) {
				return nil, model.ErrNoteNotFound
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Note, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Note, error) {
				return &model.Note{ID: "n1"}, nil
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
		srv := newNoteServer(noteRepoStub{})
		_, err := srv.List(context.Background(), &pb.ListRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.Note, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.List(authed("u1"), &pb.ListRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.Note, error) {
				return []*model.Note{{ID: "n1"}, {ID: "n2"}}, nil
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
		srv := newNoteServer(noteRepoStub{})
		_, err := srv.Update(context.Background(), &pb.UpdateRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("version conflict", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Note) (*model.Note, error) {
				return nil, model.ErrVersionConflict
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Aborted {
			t.Fatalf("code = %v, want Aborted", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Note) (*model.Note, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			update: func(_ context.Context, _ *model.User, _ *model.Note) (*model.Note, error) {
				return &model.Note{ID: "n1", Version: 2}, nil
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
		srv := newNoteServer(noteRepoStub{})
		_, err := srv.Delete(context.Background(), &pb.DeleteRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return model.ErrNoteNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return errors.New("db down") },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNoteServerChanges(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{})
		_, err := srv.Changes(context.Background(), &pb.ChangesRequest{})
		if status.Code(err) != codes.Unauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", status.Code(err))
		}
	})

	t.Run("internal error", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.NoteChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := srv.Changes(authed("u1"), &pb.ChangesRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newNoteServer(noteRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.NoteChange, error) {
				return []*model.NoteChange{{ID: "n1"}}, nil
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
