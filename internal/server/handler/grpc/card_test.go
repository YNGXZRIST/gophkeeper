package grpc

import (
	"context"
	"testing"

	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/card/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cardRepoStub struct {
	getByUser func(context.Context, *model.User, string, int, int) ([]*model.Card, error)
	getByID   func(context.Context, *model.User, string) (*model.Card, error)
	create    func(context.Context, *model.User, *model.Card) (*model.Card, error)
	update    func(context.Context, *model.User, *model.Card) (*model.Card, error)
	del       func(context.Context, *model.User, string) error
}

func (s cardRepoStub) GetByUser(ctx context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Card, error) {
	return s.getByUser(ctx, u, lastID, limit, offset)
}
func (s cardRepoStub) GetByID(ctx context.Context, u *model.User, id string) (*model.Card, error) {
	return s.getByID(ctx, u, id)
}
func (s cardRepoStub) Create(ctx context.Context, u *model.User, c *model.Card) (*model.Card, error) {
	return s.create(ctx, u, c)
}
func (s cardRepoStub) Update(ctx context.Context, u *model.User, c *model.Card) (*model.Card, error) {
	return s.update(ctx, u, c)
}
func (s cardRepoStub) Delete(ctx context.Context, u *model.User, id string) error {
	return s.del(ctx, u, id)
}

func newCardServer(repo cardRepoStub) *CardServer {
	return NewCardServer(CardServerProp{
		Service: service.NewCardService(repo),
		Logger:  zap.NewNop(),
	})
}

func authed(userID string) context.Context {
	return authctx.ContextWithUserID(context.Background(), userID)
}

func TestCardServerAdd(t *testing.T) {
	srv := newCardServer(cardRepoStub{
		create: func(_ context.Context, _ *model.User, _ *model.Card) (*model.Card, error) {
			return &model.Card{ID: "c1", Version: 1}, nil
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
		srv := newCardServer(cardRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Card, error) {
				return nil, model.ErrCardNotFound
			},
		})
		_, err := srv.Get(authed("u1"), &pb.GetRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newCardServer(cardRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Card, error) {
				return &model.Card{ID: "c1"}, nil
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
	srv := newCardServer(cardRepoStub{
		getByUser: func(_ context.Context, _ *model.User, _ string, _, _ int) ([]*model.Card, error) {
			return []*model.Card{{ID: "c1"}, {ID: "c2"}}, nil
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
	srv := newCardServer(cardRepoStub{
		update: func(_ context.Context, _ *model.User, _ *model.Card) (*model.Card, error) {
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
		srv := newCardServer(cardRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return model.ErrCardNotFound },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("code = %v, want NotFound", status.Code(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := newCardServer(cardRepoStub{
			del: func(_ context.Context, _ *model.User, _ string) error { return nil },
		})
		_, err := srv.Delete(authed("u1"), &pb.DeleteRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
