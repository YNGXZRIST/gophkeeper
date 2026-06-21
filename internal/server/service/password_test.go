package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
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
	return s.changes(ctx, u, since)
}

func TestPasswordServiceAdd(t *testing.T) {
	want := &model.Password{ID: "p1"}
	var gotUser *model.User
	var gotPass *model.Password
	svc := NewPasswordService(passwordRepoStub{
		create: func(_ context.Context, u *model.User, p *model.Password) (*model.Password, error) {
			gotUser, gotPass = u, p
			return want, nil
		},
	})

	got, err := svc.Add(context.Background(), "u1", "p1", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("password = %v, want %v", got, want)
	}
	if gotUser.ID != "u1" || gotPass.ID != "p1" || string(gotPass.Data) != "data" {
		t.Errorf("args = (%q,%q,%q), want (u1,p1,data)", gotUser.ID, gotPass.ID, gotPass.Data)
	}
}

func TestPasswordServiceList(t *testing.T) {
	want := []*model.Password{{ID: "p1"}, {ID: "p2"}}
	var gotUser *model.User
	var gotLast string
	var gotLimit, gotOffset int
	svc := NewPasswordService(passwordRepoStub{
		getByUser: func(_ context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Password, error) {
			gotUser, gotLast, gotLimit, gotOffset = u, lastID, limit, offset
			return want, nil
		},
	})

	got, err := svc.List(context.Background(), "u1", "p0", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if gotUser.ID != "u1" || gotLast != "p0" || gotLimit != 10 || gotOffset != 5 {
		t.Errorf("args = (%q,%q,%d,%d), want (u1,p0,10,5)", gotUser.ID, gotLast, gotLimit, gotOffset)
	}
}

func TestPasswordServiceGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := &model.Password{ID: "p1"}
		svc := NewPasswordService(passwordRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Password, error) { return want, nil },
		})
		got, err := svc.Get(context.Background(), "u1", "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("password = %v, want %v", got, want)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := NewPasswordService(passwordRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Password, error) {
				return nil, model.ErrPasswordNotFound
			},
		})
		_, err := svc.Get(context.Background(), "u1", "missing")
		if !errors.Is(err, model.ErrPasswordNotFound) {
			t.Fatalf("err = %v, want ErrPasswordNotFound", err)
		}
	})
}

func TestPasswordServiceUpdate(t *testing.T) {
	want := &model.Password{ID: "p1", Version: 2}
	var gotPass *model.Password
	svc := NewPasswordService(passwordRepoStub{
		update: func(_ context.Context, _ *model.User, p *model.Password) (*model.Password, error) {
			gotPass = p
			return want, nil
		},
	})

	got, err := svc.Update(context.Background(), "u1", "p1", []byte("new"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("password = %v, want %v", got, want)
	}
	if gotPass.ID != "p1" || gotPass.Version != 1 || string(gotPass.Data) != "new" {
		t.Errorf("password passed = %+v, want id=p1 version=1 data=new", gotPass)
	}
}

func TestPasswordServiceDelete(t *testing.T) {
	var gotUser *model.User
	var gotID string
	svc := NewPasswordService(passwordRepoStub{
		del: func(_ context.Context, u *model.User, id string) error {
			gotUser, gotID = u, id
			return nil
		},
	})

	if err := svc.Delete(context.Background(), "u1", "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != "u1" || gotID != "p1" {
		t.Errorf("args = (%q,%q), want (u1,p1)", gotUser.ID, gotID)
	}
}

func TestPasswordServiceChanges(t *testing.T) {
	t.Run("empty since", func(t *testing.T) {
		var gotSince time.Time
		svc := NewPasswordService(passwordRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.PasswordChange, error) {
				gotSince = since
				return []*model.PasswordChange{{ID: "p1"}}, nil
			},
		})
		got, err := svc.Changes(context.Background(), "u1", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotSince.IsZero() {
			t.Errorf("since = %v, want zero", gotSince)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
	})

	t.Run("valid since", func(t *testing.T) {
		want := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
		var gotSince time.Time
		svc := NewPasswordService(passwordRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.PasswordChange, error) {
				gotSince = since
				return nil, nil
			},
		})
		_, err := svc.Changes(context.Background(), "u1", want.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotSince.Equal(want) {
			t.Errorf("since = %v, want %v", gotSince, want)
		}
	})

	t.Run("invalid since", func(t *testing.T) {
		svc := NewPasswordService(passwordRepoStub{})
		_, err := svc.Changes(context.Background(), "u1", "not-a-time")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := NewPasswordService(passwordRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.PasswordChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := svc.Changes(context.Background(), "u1", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
