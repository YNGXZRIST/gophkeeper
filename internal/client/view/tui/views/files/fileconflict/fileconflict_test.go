package fileconflict

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	v := vault.New()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := v.UseDEK(key); err != nil {
		t.Fatalf("UseDEK: %v", err)
	}
	return v
}

func blob(t *testing.T, v *vault.Vault, meta clientmodel.FileMeta) []byte {
	t.Helper()
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b, err := v.Encrypt(raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return b
}

type fakeRepo struct {
	rows       []repository.ConflictRow
	listErr    error
	keepErr    error
	takeErr    error
	keepCalled string
	takeCalled string
}

func (r *fakeRepo) ListConflicts(_ context.Context) ([]repository.ConflictRow, error) {
	return r.rows, r.listErr
}
func (r *fakeRepo) ResolveKeepMine(_ context.Context, id string) error {
	r.keepCalled = id
	return r.keepErr
}
func (r *fakeRepo) ResolveTakeServer(_ context.Context, id string) error {
	r.takeCalled = id
	return r.takeErr
}

type fakeSync struct{ err error }

func (s fakeSync) SyncAll(_ context.Context) error { return s.err }

func newWithRows(t *testing.T, v *vault.Vault, repo *fakeRepo) model {
	t.Helper()
	return New(Prop{Vault: v, Repo: repo, Sync: fakeSync{}}).(model)
}

func TestInitRunSync(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Sync: fakeSync{}}).(model)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil")
	}
	if _, ok := cmd().(syncedMsg); !ok {
		t.Fatal("Init cmd should yield syncedMsg")
	}
}

func TestRunSyncNilSyncer(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}}).(model)
	msg := m.runSync()()
	s, ok := msg.(syncedMsg)
	if !ok || s.err != nil {
		t.Fatalf("nil syncer should yield empty syncedMsg, got %#v", msg)
	}
}

func TestSyncedThenLoad(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "c-1", Local: blob(t, v, clientmodel.FileMeta{Name: "mine.txt"}), Server: blob(t, v, clientmodel.FileMeta{Name: "theirs.txt"})},
	}}
	m := newWithRows(t, v, repo)

	m2, cmd := m.Update(syncedMsg{})
	if cmd == nil {
		t.Fatal("syncedMsg should trigger load cmd")
	}
	loaded, ok := cmd().(loadedMsg)
	if !ok || loaded.err != nil {
		t.Fatalf("expected loadedMsg, got %#v", cmd())
	}
	if len(loaded.items) != 1 || loaded.items[0].local.Name != "mine.txt" {
		t.Fatalf("decoded items wrong: %#v", loaded.items)
	}

	m3, _ := m2.Update(loaded)
	mm := m3.(model)
	if mm.loading || len(mm.items) != 1 {
		t.Fatalf("after load: loading=%v items=%d", mm.loading, len(mm.items))
	}
	if !strings.Contains(mm.View().Content, "MINE") {
		t.Error("view should show MINE/SERVER block")
	}
	if !strings.Contains(mm.View().Content, "SERVER") {
		t.Error("view should show SERVER line")
	}
}

func TestSyncError(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Sync: fakeSync{err: errors.New("net")}}).(model)
	m2, _ := m.Update(syncedMsg{err: errors.New("net")})
	if !strings.Contains(m2.(model).status, "Sync error") {
		t.Error("expected sync error status")
	}
}

func TestLoadError(t *testing.T) {
	m := newWithRows(t, testVault(t), &fakeRepo{})
	m2, _ := m.Update(loadedMsg{err: errors.New("db")})
	mm := m2.(model)
	if mm.loading {
		t.Error("loading should be cleared")
	}
	if !strings.Contains(mm.status, "Load error") {
		t.Error("expected load error status")
	}
	if !strings.Contains(mm.View().Content, "Load error") {
		t.Error("view should render status")
	}
}

func TestKeepMineResolve(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{{ID: "c-1", Local: blob(t, v, clientmodel.FileMeta{Name: "a"}), Server: blob(t, v, clientmodel.FileMeta{Name: "b"})}}}
	m := newWithRows(t, v, repo)
	m.items = []item{{id: "c-1"}}

	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if !m2.(model).loading {
		t.Error("resolve should set loading")
	}
	if cmd == nil {
		t.Fatal("m key should trigger resolve cmd")
	}
	if _, ok := cmd().(resolvedMsg); !ok {
		t.Fatal("expected resolvedMsg")
	}
	if repo.keepCalled != "c-1" {
		t.Errorf("ResolveKeepMine id = %q", repo.keepCalled)
	}
}

func TestTakeServerResolve(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{}
	m := newWithRows(t, v, repo)
	m.items = []item{{id: "c-1"}}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if cmd == nil {
		t.Fatal("s key should trigger resolve cmd")
	}
	cmd()
	if repo.takeCalled != "c-1" {
		t.Errorf("ResolveTakeServer id = %q", repo.takeCalled)
	}
}

func TestResolveErrorAndOK(t *testing.T) {
	m := newWithRows(t, testVault(t), &fakeRepo{})
	m2, cmd := m.Update(resolvedMsg{err: errors.New("x")})
	if cmd != nil {
		t.Error("resolve error should not trigger sync")
	}
	if !strings.Contains(m2.(model).status, "Resolve error") {
		t.Error("expected resolve error status")
	}
	m3, cmd2 := m.Update(resolvedMsg{})
	if !m3.(model).loading || cmd2 == nil {
		t.Fatal("successful resolve should set loading and batch sync")
	}
}

func TestNavigationKeys(t *testing.T) {
	m := newWithRows(t, testVault(t), &fakeRepo{})
	m.loading = false
	m.items = []item{{id: "a"}, {id: "b"}, {id: "c"}}

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.(model).selected != 1 {
		t.Errorf("down: selected = %d", m2.(model).selected)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m3.(model).selected != 0 {
		t.Errorf("up: selected = %d", m3.(model).selected)
	}
	m4, _ := m3.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m4.(model).selected != 0 {
		t.Errorf("up at top: selected = %d", m4.(model).selected)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should produce Back cmd")
	}
}

func TestResolveKeysNoItems(t *testing.T) {
	m := newWithRows(t, testVault(t), &fakeRepo{})
	m.loading = false
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"}); cmd != nil {
		t.Error("m with no items should be a no-op")
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"}); cmd != nil {
		t.Error("s with no items should be a no-op")
	}
}

func TestViewLoadingAndEmpty(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Sync: fakeSync{}}).(model)
	if !strings.Contains(m.View().Content, "Syncing") {
		t.Error("loading view should show Syncing")
	}
	m.loading = false
	if !strings.Contains(m.View().Content, "No conflicts") {
		t.Error("empty view should show No conflicts")
	}
}

func TestDecodeCorrupt(t *testing.T) {
	v := testVault(t)
	if got := decode(v, []byte("garbage")); got != (clientmodel.FileMeta{}) {
		t.Errorf("decode corrupt should be zero value, got %#v", got)
	}
	bad, err := v.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got := decode(v, bad); got != (clientmodel.FileMeta{}) {
		t.Errorf("decode bad json should be zero value, got %#v", got)
	}
}

func TestLineHelper(t *testing.T) {
	if line(clientmodel.FileMeta{}) != "—" {
		t.Error("empty name should be em dash")
	}
	if line(clientmodel.FileMeta{Name: "a\nb"}) != "a b" {
		t.Error("newlines should be replaced")
	}
}
