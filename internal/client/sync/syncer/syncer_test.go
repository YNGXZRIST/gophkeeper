package syncer

import (
	"context"
	"errors"
	"testing"
)

type fakeRepo struct {
	rows       map[string]Row
	cursor     string
	conflicted map[string]bool
	serverBlob map[string][]byte
	syncedVer  map[string]int64
	hardDel    map[string]bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		rows:       map[string]Row{},
		conflicted: map[string]bool{},
		serverBlob: map[string][]byte{},
		syncedVer:  map[string]int64{},
		hardDel:    map[string]bool{},
	}
}

func (f *fakeRepo) ListDirty(ctx context.Context) ([]Row, error) {
	var out []Row
	for _, r := range f.rows {
		if r.Dirty && !f.conflicted[r.ID] {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRepo) Get(ctx context.Context, id string) (Row, bool, error) {
	r, ok := f.rows[id]
	return r, ok, nil
}

func (f *fakeRepo) Upsert(ctx context.Context, id string, data []byte, version int64) error {
	f.rows[id] = Row{ID: id, Data: data, Version: version, BaseVersion: version}
	return nil
}

func (f *fakeRepo) HardDelete(ctx context.Context, id string) error {
	delete(f.rows, id)
	f.hardDel[id] = true
	return nil
}

func (f *fakeRepo) MarkSynced(ctx context.Context, id string, version int64) error {
	r := f.rows[id]
	r.Dirty = false
	r.Version = version
	r.BaseVersion = version
	f.rows[id] = r
	f.syncedVer[id] = version
	return nil
}

func (f *fakeRepo) MarkConflict(ctx context.Context, id string, serverBlob []byte, serverVersion int64) error {
	f.conflicted[id] = true
	f.serverBlob[id] = serverBlob
	return nil
}

func (f *fakeRepo) LastSyncedAt(ctx context.Context) (string, error) { return f.cursor, nil }

func (f *fakeRepo) SetLastSyncedAt(ctx context.Context, cursor string) error {
	f.cursor = cursor
	return nil
}

type fakeClient struct {
	changes    []Change
	cursor     string
	changesErr error

	created map[string][]byte
	updated map[string][]byte
	deleted map[string]bool

	createVer int64
	updateVer int64
	createErr error
	updateErr error
	deleteErr error
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		created:   map[string][]byte{},
		updated:   map[string][]byte{},
		deleted:   map[string]bool{},
		createVer: 1,
		updateVer: 1,
	}
}

func (c *fakeClient) Changes(ctx context.Context, since string) ([]Change, string, error) {
	if c.changesErr != nil {
		return nil, since, c.changesErr
	}
	cur := c.cursor
	if cur == "" {
		cur = since
	}
	return c.changes, cur, nil
}

func (c *fakeClient) Create(ctx context.Context, id string, data []byte) (int64, error) {
	if c.createErr != nil {
		return 0, c.createErr
	}
	c.created[id] = data
	return c.createVer, nil
}

func (c *fakeClient) Update(ctx context.Context, id string, data []byte, version int64) (int64, error) {
	if c.updateErr != nil {
		return 0, c.updateErr
	}
	c.updated[id] = data
	return c.updateVer, nil
}

func (c *fakeClient) Delete(ctx context.Context, id string, version int64) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	c.deleted[id] = true
	return nil
}

func TestSyncPullInsertsServerChange(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	client.changes = []Change{{ID: "n1", Data: []byte("hello"), Version: 3}}
	client.cursor = "2026-01-01T00:00:00Z"

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	got, ok := repo.rows["n1"]
	if !ok || string(got.Data) != "hello" || got.Version != 3 {
		t.Fatalf("row not upserted: %+v ok=%v", got, ok)
	}
	if repo.cursor != client.cursor {
		t.Fatalf("cursor = %q, want %q", repo.cursor, client.cursor)
	}
}

func TestSyncPullDeletesTombstone(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Version: 1, BaseVersion: 1}
	client := newFakeClient()
	client.changes = []Change{{ID: "n1", Deleted: true, Version: 2}}

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !repo.hardDel["n1"] {
		t.Fatal("expected hard delete")
	}
}

func TestSyncPullDetectsConflict(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("mine"), Version: 2, BaseVersion: 1, Dirty: true}
	client := newFakeClient()
	client.changes = []Change{{ID: "n1", Data: []byte("server"), Version: 5}}

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !repo.conflicted["n1"] {
		t.Fatal("expected conflict")
	}
	if string(repo.serverBlob["n1"]) != "server" {
		t.Fatalf("server blob = %q", repo.serverBlob["n1"])
	}
	if string(repo.rows["n1"].Data) != "mine" {
		t.Fatal("local data must be kept on conflict")
	}
}

func TestSyncPullDirtyNoConflictWhenBaseMatches(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("mine"), Version: 3, BaseVersion: 2, Dirty: true}
	client := newFakeClient()
	client.changes = []Change{{ID: "n1", Data: []byte("server"), Version: 2}}

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if repo.conflicted["n1"] {
		t.Fatal("must not flag conflict when server version equals base")
	}
	if string(repo.rows["n1"].Data) != "mine" {
		t.Fatal("local dirty data must be kept")
	}
}

func TestSyncPushCreate(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("new"), Version: 1, BaseVersion: 0, Dirty: true}
	client := newFakeClient()
	client.createVer = 7

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if string(client.created["n1"]) != "new" {
		t.Fatal("expected create call")
	}
	if repo.syncedVer["n1"] != 7 {
		t.Fatalf("synced version = %d, want 7", repo.syncedVer["n1"])
	}
}

func TestSyncPushUpdate(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("edit"), Version: 4, BaseVersion: 3, Dirty: true}
	client := newFakeClient()
	client.updateVer = 4

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if string(client.updated["n1"]) != "edit" {
		t.Fatal("expected update call")
	}
	if repo.syncedVer["n1"] != 4 {
		t.Fatalf("synced version = %d, want 4", repo.syncedVer["n1"])
	}
}

func TestSyncPushDelete(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Version: 2, BaseVersion: 2, Dirty: true, Deleted: true}
	client := newFakeClient()

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !client.deleted["n1"] {
		t.Fatal("expected delete call")
	}
	if !repo.hardDel["n1"] {
		t.Fatal("expected hard delete after push delete")
	}
}

func TestSyncPushVersionConflict(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("edit"), Version: 4, BaseVersion: 3, Dirty: true}
	client := newFakeClient()
	client.updateErr = ErrVersionConflict

	if err := New(repo, client).Sync(context.Background()); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !repo.conflicted["n1"] {
		t.Fatal("expected conflict flagged on push version conflict")
	}
}

func TestSyncPullErrorPropagates(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	client.changesErr = errors.New("boom")

	if err := New(repo, client).Sync(context.Background()); err == nil {
		t.Fatal("expected error from pull")
	}
}

func TestSyncPushErrorPropagates(t *testing.T) {
	repo := newFakeRepo()
	repo.rows["n1"] = Row{ID: "n1", Data: []byte("edit"), Version: 4, BaseVersion: 3, Dirty: true}
	client := newFakeClient()
	client.updateErr = errors.New("boom")

	if err := New(repo, client).Sync(context.Background()); err == nil {
		t.Fatal("expected error from push")
	}
}
