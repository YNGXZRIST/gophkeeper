package service

import (
	"context"
	"errors"
	"testing"

	"gophkeeper/internal/server/model"
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

func TestCardServiceAdd(t *testing.T) {
	want := &model.Card{ID: "c1"}
	var gotUser *model.User
	var gotCard *model.Card
	svc := NewCardService(cardRepoStub{
		create: func(_ context.Context, u *model.User, c *model.Card) (*model.Card, error) {
			gotUser, gotCard = u, c
			return want, nil
		},
	})

	got, err := svc.Add(context.Background(), "u1", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("card = %v, want %v", got, want)
	}
	if gotUser.ID != "u1" {
		t.Errorf("user ID = %q, want u1", gotUser.ID)
	}
	if string(gotCard.Data) != "data" {
		t.Errorf("data = %q, want data", gotCard.Data)
	}
}

func TestCardServiceList(t *testing.T) {
	want := []*model.Card{{ID: "c1"}, {ID: "c2"}}
	var gotUser *model.User
	var gotLast string
	var gotLimit, gotOffset int
	svc := NewCardService(cardRepoStub{
		getByUser: func(_ context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Card, error) {
			gotUser, gotLast, gotLimit, gotOffset = u, lastID, limit, offset
			return want, nil
		},
	})

	got, err := svc.List(context.Background(), "u1", "c0", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if gotUser.ID != "u1" || gotLast != "c0" || gotLimit != 10 || gotOffset != 5 {
		t.Errorf("args = (%q,%q,%d,%d), want (u1,c0,10,5)", gotUser.ID, gotLast, gotLimit, gotOffset)
	}
}

func TestCardServiceGet(t *testing.T) {
	svc := NewCardService(cardRepoStub{
		getByID: func(_ context.Context, _ *model.User, _ string) (*model.Card, error) {
			return nil, model.ErrCardNotFound
		},
	})

	_, err := svc.Get(context.Background(), "u1", "missing")
	if !errors.Is(err, model.ErrCardNotFound) {
		t.Fatalf("err = %v, want ErrCardNotFound", err)
	}
}

func TestCardServiceUpdate(t *testing.T) {
	want := &model.Card{ID: "c1", Version: 2}
	var gotCard *model.Card
	svc := NewCardService(cardRepoStub{
		update: func(_ context.Context, _ *model.User, c *model.Card) (*model.Card, error) {
			gotCard = c
			return want, nil
		},
	})

	got, err := svc.Update(context.Background(), "u1", "c1", []byte("new"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("card = %v, want %v", got, want)
	}
	if gotCard.ID != "c1" || gotCard.Version != 1 || string(gotCard.Data) != "new" {
		t.Errorf("card passed = %+v, want id=c1 version=1 data=new", gotCard)
	}
}

func TestCardServiceDelete(t *testing.T) {
	var gotUser *model.User
	var gotID string
	svc := NewCardService(cardRepoStub{
		del: func(_ context.Context, u *model.User, id string) error {
			gotUser, gotID = u, id
			return nil
		},
	})

	if err := svc.Delete(context.Background(), "u1", "c1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != "u1" || gotID != "c1" {
		t.Errorf("args = (%q,%q), want (u1,c1)", gotUser.ID, gotID)
	}
}
