package syncclient

import (
	"context"
	"errors"
	"testing"

	"gophkeeper/internal/client/sync/syncer"
)

type fakeRepo struct {
	listErr   error
	listCalls *int
}

func (f *fakeRepo) ListDirty(context.Context) ([]syncer.Row, error) {
	if f.listCalls != nil {
		*f.listCalls++
	}
	return nil, f.listErr
}
func (f *fakeRepo) Get(context.Context, string) (syncer.Row, bool, error) {
	return syncer.Row{}, false, nil
}
func (f *fakeRepo) Upsert(context.Context, string, []byte, int64) error       { return nil }
func (f *fakeRepo) HardDelete(context.Context, string) error                  { return nil }
func (f *fakeRepo) MarkSynced(context.Context, string, int64) error           { return nil }
func (f *fakeRepo) MarkConflict(context.Context, string, []byte, int64) error { return nil }
func (f *fakeRepo) LastSyncedAt(context.Context) (string, error)              { return "", nil }
func (f *fakeRepo) SetLastSyncedAt(context.Context, string) error             { return nil }

type fakeClient struct{}

func (fakeClient) Changes(_ context.Context, since string) ([]syncer.Change, string, error) {
	return nil, since, nil
}
func (fakeClient) Create(context.Context, string, []byte) (int64, error)        { return 1, nil }
func (fakeClient) Update(context.Context, string, []byte, int64) (int64, error) { return 1, nil }
func (fakeClient) Delete(context.Context, string, int64) error                  { return nil }

func TestSyncAllRunsAll(t *testing.T) {
	var c1, c2 int
	s1 := syncer.New(&fakeRepo{listCalls: &c1}, fakeClient{})
	s2 := syncer.New(&fakeRepo{listCalls: &c2}, fakeClient{})

	if err := New(s1, s2).SyncAll(context.Background()); err != nil {
		t.Fatalf("sync all: %v", err)
	}
	if c1 != 1 || c2 != 1 {
		t.Fatalf("each syncer must run once: c1=%d c2=%d", c1, c2)
	}
}

func TestSyncAllReturnsFirstError(t *testing.T) {
	boom := errors.New("boom")
	var c2 int
	s1 := syncer.New(&fakeRepo{listErr: boom}, fakeClient{})
	s2 := syncer.New(&fakeRepo{listCalls: &c2}, fakeClient{})

	err := New(s1, s2).SyncAll(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if c2 != 0 {
		t.Fatalf("second syncer must not run after error, c2=%d", c2)
	}
}

func TestSyncAllEmpty(t *testing.T) {
	if err := New().SyncAll(context.Background()); err != nil {
		t.Fatalf("empty pool: %v", err)
	}
}
