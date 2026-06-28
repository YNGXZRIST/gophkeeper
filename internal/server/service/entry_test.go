package service

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"gophkeeper/internal/server/model"
)

type entryRepoStub struct {
	getByUser func(context.Context, string, string, int, int) ([]*model.Entry, error)
	getByID   func(context.Context, string, string) (*model.Entry, error)
	create    func(context.Context, string, *model.Entry) (*model.Entry, error)
	update    func(context.Context, string, *model.Entry) (*model.Entry, error)
	del       func(context.Context, string, string) error
	changes   func(context.Context, string, time.Time) ([]*model.EntryChange, error)
}

func (s entryRepoStub) GetByUser(ctx context.Context, uid, lastID string, limit, offset int) iter.Seq2[*model.Entry, error] {
	return sliceSeq(s.getByUser(ctx, uid, lastID, limit, offset))
}

// sliceSeq adapts a slice-or-error result into the streaming iterator the
// repository now returns, so the table-driven cases keep returning plain slices.
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

func (s entryRepoStub) GetByID(ctx context.Context, uid, id string) (*model.Entry, error) {
	return s.getByID(ctx, uid, id)
}

func (s entryRepoStub) Create(ctx context.Context, uid string, e *model.Entry) (*model.Entry, error) {
	return s.create(ctx, uid, e)
}

func (s entryRepoStub) Update(ctx context.Context, uid string, e *model.Entry) (*model.Entry, error) {
	return s.update(ctx, uid, e)
}

func (s entryRepoStub) Delete(ctx context.Context, uid, id string) error {
	return s.del(ctx, uid, id)
}

func (s entryRepoStub) Changes(ctx context.Context, uid string, since time.Time) iter.Seq2[*model.EntryChange, error] {
	if s.changes == nil {
		return sliceSeq[model.EntryChange](nil, nil)
	}
	return sliceSeq(s.changes(ctx, uid, since))
}

func TestEntryServiceAdd(t *testing.T) {
	want := &model.Entry{ID: "e1"}
	var gotUID string
	var gotEntry *model.Entry
	svc := NewEntryService(entryRepoStub{
		create: func(_ context.Context, uid string, e *model.Entry) (*model.Entry, error) {
			gotUID, gotEntry = uid, e
			return want, nil
		},
	})

	got, err := svc.Add(context.Background(), "u1", "e1", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("entry = %v, want %v", got, want)
	}
	if gotUID != "u1" {
		t.Errorf("uid = %q, want u1", gotUID)
	}
	if string(gotEntry.Data) != "data" {
		t.Errorf("data = %q, want data", gotEntry.Data)
	}
}

func TestEntryServiceList(t *testing.T) {
	want := []*model.Entry{{ID: "e1"}, {ID: "e2"}}
	var gotUID, gotLast string
	var gotLimit, gotOffset int
	svc := NewEntryService(entryRepoStub{
		getByUser: func(_ context.Context, uid, lastID string, limit, offset int) ([]*model.Entry, error) {
			gotUID, gotLast, gotLimit, gotOffset = uid, lastID, limit, offset
			return want, nil
		},
	})

	got, err := svc.List(context.Background(), "u1", "e0", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if gotUID != "u1" || gotLast != "e0" || gotLimit != 10 || gotOffset != 5 {
		t.Errorf("args = (%q,%q,%d,%d), want (u1,e0,10,5)", gotUID, gotLast, gotLimit, gotOffset)
	}
}

func TestEntryServiceGet(t *testing.T) {
	svc := NewEntryService(entryRepoStub{
		getByID: func(_ context.Context, _ string, _ string) (*model.Entry, error) {
			return nil, model.ErrEntryNotFound
		},
	})

	_, err := svc.Get(context.Background(), "u1", "missing")
	if !errors.Is(err, model.ErrEntryNotFound) {
		t.Fatalf("err = %v, want ErrEntryNotFound", err)
	}
}

func TestEntryServiceUpdate(t *testing.T) {
	want := &model.Entry{ID: "e1", Version: 2}
	var gotEntry *model.Entry
	svc := NewEntryService(entryRepoStub{
		update: func(_ context.Context, _ string, e *model.Entry) (*model.Entry, error) {
			gotEntry = e
			return want, nil
		},
	})

	got, err := svc.Update(context.Background(), "u1", "e1", []byte("new"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("entry = %v, want %v", got, want)
	}
	if gotEntry.ID != "e1" || gotEntry.Version != 1 || string(gotEntry.Data) != "new" {
		t.Errorf("entry passed = %+v, want id=e1 version=1 data=new", gotEntry)
	}
}

func TestEntryServiceDelete(t *testing.T) {
	var gotUID, gotID string
	svc := NewEntryService(entryRepoStub{
		del: func(_ context.Context, uid, id string) error {
			gotUID, gotID = uid, id
			return nil
		},
	})

	if err := svc.Delete(context.Background(), "u1", "e1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUID != "u1" || gotID != "e1" {
		t.Errorf("args = (%q,%q), want (u1,e1)", gotUID, gotID)
	}
}

func TestEntryServiceChanges(t *testing.T) {
	want := []*model.EntryChange{{ID: "e1"}}
	var gotUID string
	var gotSince time.Time
	svc := NewEntryService(entryRepoStub{
		changes: func(_ context.Context, uid string, since time.Time) ([]*model.EntryChange, error) {
			gotUID, gotSince = uid, since
			return want, nil
		},
	})

	got, err := svc.Changes(context.Background(), "u1", "2024-01-02T03:04:05Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if gotUID != "u1" || gotSince.IsZero() {
		t.Errorf("args = (%q, %v), want (u1, non-zero)", gotUID, gotSince)
	}
}
