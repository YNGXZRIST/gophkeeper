package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
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
	return s.changes(ctx, u, since)
}

func TestNoteServiceAdd(t *testing.T) {
	want := &model.Note{ID: "n1"}
	var gotUser *model.User
	var gotNote *model.Note
	svc := NewNoteService(noteRepoStub{
		create: func(_ context.Context, u *model.User, n *model.Note) (*model.Note, error) {
			gotUser, gotNote = u, n
			return want, nil
		},
	})

	got, err := svc.Add(context.Background(), "u1", "n1", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("note = %v, want %v", got, want)
	}
	if gotUser.ID != "u1" || gotNote.ID != "n1" || string(gotNote.Data) != "data" {
		t.Errorf("args = (%q,%q,%q), want (u1,n1,data)", gotUser.ID, gotNote.ID, gotNote.Data)
	}
}

func TestNoteServiceList(t *testing.T) {
	want := []*model.Note{{ID: "n1"}, {ID: "n2"}}
	var gotUser *model.User
	var gotLast string
	var gotLimit, gotOffset int
	svc := NewNoteService(noteRepoStub{
		getByUser: func(_ context.Context, u *model.User, lastID string, limit, offset int) ([]*model.Note, error) {
			gotUser, gotLast, gotLimit, gotOffset = u, lastID, limit, offset
			return want, nil
		},
	})

	got, err := svc.List(context.Background(), "u1", "n0", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if gotUser.ID != "u1" || gotLast != "n0" || gotLimit != 10 || gotOffset != 5 {
		t.Errorf("args = (%q,%q,%d,%d), want (u1,n0,10,5)", gotUser.ID, gotLast, gotLimit, gotOffset)
	}
}

func TestNoteServiceGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		want := &model.Note{ID: "n1"}
		svc := NewNoteService(noteRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Note, error) { return want, nil },
		})
		got, err := svc.Get(context.Background(), "u1", "n1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("note = %v, want %v", got, want)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := NewNoteService(noteRepoStub{
			getByID: func(_ context.Context, _ *model.User, _ string) (*model.Note, error) {
				return nil, model.ErrNoteNotFound
			},
		})
		_, err := svc.Get(context.Background(), "u1", "missing")
		if !errors.Is(err, model.ErrNoteNotFound) {
			t.Fatalf("err = %v, want ErrNoteNotFound", err)
		}
	})
}

func TestNoteServiceUpdate(t *testing.T) {
	want := &model.Note{ID: "n1", Version: 2}
	var gotNote *model.Note
	svc := NewNoteService(noteRepoStub{
		update: func(_ context.Context, _ *model.User, n *model.Note) (*model.Note, error) {
			gotNote = n
			return want, nil
		},
	})

	got, err := svc.Update(context.Background(), "u1", "n1", []byte("new"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("note = %v, want %v", got, want)
	}
	if gotNote.ID != "n1" || gotNote.Version != 1 || string(gotNote.Data) != "new" {
		t.Errorf("note passed = %+v, want id=n1 version=1 data=new", gotNote)
	}
}

func TestNoteServiceDelete(t *testing.T) {
	var gotUser *model.User
	var gotID string
	svc := NewNoteService(noteRepoStub{
		del: func(_ context.Context, u *model.User, id string) error {
			gotUser, gotID = u, id
			return nil
		},
	})

	if err := svc.Delete(context.Background(), "u1", "n1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser.ID != "u1" || gotID != "n1" {
		t.Errorf("args = (%q,%q), want (u1,n1)", gotUser.ID, gotID)
	}
}

func TestNoteServiceChanges(t *testing.T) {
	t.Run("empty since", func(t *testing.T) {
		var gotSince time.Time
		svc := NewNoteService(noteRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.NoteChange, error) {
				gotSince = since
				return []*model.NoteChange{{ID: "n1"}}, nil
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
		svc := NewNoteService(noteRepoStub{
			changes: func(_ context.Context, _ *model.User, since time.Time) ([]*model.NoteChange, error) {
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
		svc := NewNoteService(noteRepoStub{})
		_, err := svc.Changes(context.Background(), "u1", "not-a-time")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := NewNoteService(noteRepoStub{
			changes: func(_ context.Context, _ *model.User, _ time.Time) ([]*model.NoteChange, error) {
				return nil, errors.New("db down")
			},
		})
		_, err := svc.Changes(context.Background(), "u1", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
