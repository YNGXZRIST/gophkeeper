package grpc

import (
	"context"
	"iter"
	"testing"
	"time"

	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/card/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// entryStoreStub is the shared service-store stub: the card, note and password
// handler tests all build their servers from it via service.NewEntryService.
type entryStoreStub struct {
	getByUser func(context.Context, string, string, int, int) ([]*model.Entry, error)
	getByID   func(context.Context, string, string) (*model.Entry, error)
	create    func(context.Context, string, *model.Entry) (*model.Entry, error)
	update    func(context.Context, string, *model.Entry) (*model.Entry, error)
	del       func(context.Context, string, string) error
	changes   func(context.Context, string, time.Time) ([]*model.EntryChange, error)
}

func (s entryStoreStub) GetByUser(ctx context.Context, uid, lastID string, limit, offset int) iter.Seq2[*model.Entry, error] {
	return sliceSeq(s.getByUser(ctx, uid, lastID, limit, offset))
}

// sliceSeq adapts a slice-or-error result into the streaming iterator the
// store now returns, so the stub fields stay plain slices.
func sliceSeq[T any](items []*T, err error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		if err != nil {
			yield(nil, err)
			return
		}
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}
func (s entryStoreStub) GetByID(ctx context.Context, uid, id string) (*model.Entry, error) {
	return s.getByID(ctx, uid, id)
}
func (s entryStoreStub) Create(ctx context.Context, uid string, e *model.Entry) (*model.Entry, error) {
	return s.create(ctx, uid, e)
}
func (s entryStoreStub) Update(ctx context.Context, uid string, e *model.Entry) (*model.Entry, error) {
	return s.update(ctx, uid, e)
}
func (s entryStoreStub) Delete(ctx context.Context, uid, id string) error {
	return s.del(ctx, uid, id)
}
func (s entryStoreStub) Changes(ctx context.Context, uid string, since time.Time) iter.Seq2[*model.EntryChange, error] {
	if s.changes == nil {
		return sliceSeq[model.EntryChange](nil, nil)
	}
	return sliceSeq(s.changes(ctx, uid, since))
}

func authed(userID string) context.Context {
	return authctx.ContextWithUserID(context.Background(), userID)
}

func newCardServer(repo entryStoreStub) *CardServer {
	return NewCardServer(CardServerProp{
		Service: service.NewEntryService(repo),
		Logger:  zap.NewNop(),
	})
}

func TestCardServerAdd(t *testing.T) {
	srv := newCardServer(entryStoreStub{
		create: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
			return &model.Entry{ID: "c1", Version: 1}, nil
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
		if resp.GetCard().GetId() != "c1" {
			t.Fatalf("id = %q, want c1", resp.GetCard().GetId())
		}
	})
}

func TestCardServerGet(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		srv := newCardServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return nil, model.ErrEntryNotFound
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newCardServer(entryStoreStub{
			getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
				return &model.Entry{ID: "c1"}, nil
			},
		})
		resp, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.GetCard().GetId() != "c1" {
			t.Fatalf("id = %q, want c1", resp.GetCard().GetId())
		}
	})
}

func TestCardServerList(t *testing.T) {
	srv := newCardServer(entryStoreStub{
		getByUser: func(_ context.Context, _ string, _ string, _, _ int) ([]*model.Entry, error) {
			return []*model.Entry{{ID: "c1"}, {ID: "c2"}}, nil
		},
	})
	resp, err := srv.List(authed("u1"), &pb.ListRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetCards()) != 2 {
		t.Fatalf("cards = %d, want 2", len(resp.GetCards()))
	}
}

func TestCardServerUpdate(t *testing.T) {
	srv := newCardServer(entryStoreStub{
		update: func(_ context.Context, _ string, _ *model.Entry) (*model.Entry, error) {
			return nil, model.ErrVersionConflict
		},
	})
	_, err := srv.Update(authed("u1"), &pb.UpdateRequest{})
	if status.Code(err) != codes.Aborted {
		t.Fatalf("code = %v, want Aborted", status.Code(err))
	}
}

func TestCardServerDelete(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		srv := newCardServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return model.ErrEntryNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newCardServer(entryStoreStub{
			del: func(_ context.Context, _ string, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
