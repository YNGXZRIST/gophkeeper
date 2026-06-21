package cardconflict

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"reflect"
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

func enc(t *testing.T, v *vault.Vault, d clientmodel.CardData) []byte {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	blob, err := v.Encrypt(raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return blob
}

type fakeRepo struct {
	rows       []repository.ConflictRow
	listErr    error
	keepMineID string
	takeSrvID  string
	resolveErr error
}

func (f *fakeRepo) ListConflicts(context.Context) ([]repository.ConflictRow, error) {
	return f.rows, f.listErr
}
func (f *fakeRepo) ResolveKeepMine(_ context.Context, id string) error {
	f.keepMineID = id
	return f.resolveErr
}
func (f *fakeRepo) ResolveTakeServer(_ context.Context, id string) error {
	f.takeSrvID = id
	return f.resolveErr
}

type fakeSyncer struct {
	calls int
	err   error
}

func (f *fakeSyncer) SyncAll(context.Context) error {
	f.calls++
	return f.err
}

func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }
func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	return flatten(cmd())
}

func flatten(msg tea.Msg) []tea.Msg {
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, collectMsgs(c)...)
		}
		return out
	}
	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		var out []tea.Msg
		for i := 0; i < rv.Len(); i++ {
			out = append(out, collectMsgs(rv.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func twoRows(t *testing.T, v *vault.Vault) []repository.ConflictRow {
	return []repository.ConflictRow{
		{
			ID:            "id-1",
			Local:         enc(t, v, clientmodel.CardData{Number: "4111111111111111", Holder: "MINE ONE"}),
			Server:        enc(t, v, clientmodel.CardData{Number: "4222222222222222", Holder: "SERVER ONE"}),
			ServerVersion: 2,
		},
		{
			ID:            "id-2",
			Local:         enc(t, v, clientmodel.CardData{Number: "4333333333333333", Holder: "MINE TWO"}),
			Server:        enc(t, v, clientmodel.CardData{Number: "4444444444444444", Holder: "SERVER TWO"}),
			ServerVersion: 3,
		},
	}
}

func loadModel(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	syncCmd := m.Init()
	syncMsg := syncCmd()
	m, loadCmd := m.Update(syncMsg)
	if loadCmd == nil {
		t.Fatal("syncedMsg should trigger load")
	}
	m, _ = m.Update(loadCmd())
	return m
}

func TestInitRunsSyncAndLoads(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	syncer := &fakeSyncer{}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: syncer}))
	if syncer.calls != 1 {
		t.Fatalf("SyncAll calls = %d, want 1", syncer.calls)
	}
	content := m.View().Content
	if !strings.Contains(content, "MINE ONE") || !strings.Contains(content, "SERVER ONE") {
		t.Fatalf("view missing mine/server lines: %q", content)
	}
}

func TestNilSyncer(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	m := New(Prop{Vault: v, Repo: repo, Sync: nil})

	msg := m.Init()()
	if _, ok := msg.(syncedMsg); !ok {
		t.Fatalf("expected syncedMsg, got %T", msg)
	}
}

func TestSyncError(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	syncer := &fakeSyncer{err: errors.New("net down")}
	m := New(Prop{Vault: v, Repo: repo, Sync: syncer})
	m, _ = m.Update(syncedMsg{err: syncer.err})
	if !strings.Contains(m.(model).status, "Sync error") {
		t.Fatalf("status = %q", m.(model).status)
	}
}

func TestLoadError(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{listErr: errors.New("db gone")}
	m := New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}})
	m, _ = m.Update(syncedMsg{})
	m, _ = m.Update(loadedMsg{err: repo.listErr})
	mm := m.(model)
	if !strings.Contains(mm.status, "Load error") {
		t.Fatalf("status = %q", mm.status)
	}
	if mm.loading {
		t.Fatal("loading should be false after load error")
	}
}

func TestEmptyConflicts(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))
	if !strings.Contains(m.View().Content, "No conflicts") {
		t.Fatalf("view = %q", m.View().Content)
	}
}

func TestUpDownSelection(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))

	m, _ = m.Update(specialKey(tea.KeyDown))
	if m.(model).selected != 1 {
		t.Fatalf("after down selected = %d, want 1", m.(model).selected)
	}

	m, _ = m.Update(specialKey(tea.KeyDown))
	if m.(model).selected != 1 {
		t.Fatalf("after down at end selected = %d, want 1", m.(model).selected)
	}

	m, _ = m.Update(specialKey(tea.KeyUp))
	if m.(model).selected != 0 {
		t.Fatalf("after up selected = %d, want 0", m.(model).selected)
	}

	m, _ = m.Update(specialKey(tea.KeyUp))
	if m.(model).selected != 0 {
		t.Fatalf("after up at top selected = %d, want 0", m.(model).selected)
	}

	m, _ = m.Update(specialKey(tea.KeyDown))
	if !strings.Contains(m.View().Content, "MINE TWO") {
		t.Fatalf("view missing MINE TWO: %q", m.View().Content)
	}
}

func TestKeepMine(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	syncer := &fakeSyncer{}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: syncer}))

	m, cmd := m.Update(runeKey('m'))
	if cmd == nil {
		t.Fatal("m should trigger resolve")
	}
	if !m.(model).loading {
		t.Fatal("expected loading true while resolving")
	}
	msg := cmd()
	if repo.keepMineID != "id-1" {
		t.Fatalf("ResolveKeepMine id = %q, want id-1", repo.keepMineID)
	}
	rm, ok := msg.(resolvedMsg)
	if !ok || rm.err != nil {
		t.Fatalf("expected resolvedMsg ok, got %#v", msg)
	}

	_, reCmd := m.Update(rm)
	msgs := collectMsgs(reCmd)
	var sawSync, sawSyncNow bool
	for _, mm := range msgs {
		switch mm.(type) {
		case syncedMsg:
			sawSync = true
		case nav.SyncNowMsg:
			sawSyncNow = true
		}
	}
	if !sawSync || !sawSyncNow {
		t.Fatalf("expected re-sync and SyncNow, got %#v", msgs)
	}
}

func TestTakeServer(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))

	_, cmd := m.Update(runeKey('s'))
	if cmd == nil {
		t.Fatal("s should trigger resolve")
	}
	msg := cmd()
	if repo.takeSrvID != "id-1" {
		t.Fatalf("ResolveTakeServer id = %q, want id-1", repo.takeSrvID)
	}
	if _, ok := msg.(resolvedMsg); !ok {
		t.Fatalf("expected resolvedMsg, got %T", msg)
	}
}

func TestResolveError(t *testing.T) {
	v := testVault(t)
	m := New(Prop{Vault: v, Repo: &fakeRepo{}, Sync: &fakeSyncer{}})
	m, cmd := m.Update(resolvedMsg{err: errors.New("conflict gone")})
	if cmd != nil {
		t.Fatal("expected nil cmd on resolve error")
	}
	if !strings.Contains(m.(model).status, "Resolve error") {
		t.Fatalf("status = %q", m.(model).status)
	}
}

func TestKeyOnEmptyDoesNothing(t *testing.T) {
	v := testVault(t)
	m := loadModel(t, New(Prop{Vault: v, Repo: &fakeRepo{}, Sync: &fakeSyncer{}}))

	if _, cmd := m.Update(runeKey('m')); cmd != nil {
		t.Fatal("m on empty should produce nil cmd")
	}
	if _, cmd := m.Update(runeKey('s')); cmd != nil {
		t.Fatal("s on empty should produce nil cmd")
	}
}

func TestEscBack(t *testing.T) {
	v := testVault(t)
	m := loadModel(t, New(Prop{Vault: v, Repo: &fakeRepo{}, Sync: &fakeSyncer{}}))
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestLoadingView(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Sync: &fakeSyncer{}})
	if !strings.Contains(m.View().Content, "Syncing") {
		t.Fatalf("loading view = %q", m.View().Content)
	}
}

func TestDecodeCorruptBlob(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "bad", Local: []byte("garbage"), Server: []byte("garbage")},
	}}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))

	if !strings.Contains(m.View().Content, "—") {
		t.Fatalf("view = %q", m.View().Content)
	}
}

func TestSelectedResetAfterReload(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: twoRows(t, v)}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))
	m, _ = m.Update(specialKey(tea.KeyDown))

	m, _ = m.Update(loadedMsg{items: []item{{id: "only"}}})
	if m.(model).selected != 0 {
		t.Fatalf("selected = %d, want 0 after shrink", m.(model).selected)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	v := testVault(t)
	m := loadModel(t, New(Prop{Vault: v, Repo: &fakeRepo{rows: twoRows(t, v)}, Sync: &fakeSyncer{}}))
	if _, cmd := m.Update(struct{}{}); cmd != nil {
		t.Fatal("unknown msg should yield nil cmd")
	}
}

func TestMaskNumberShort(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "x", Local: enc(t, v, clientmodel.CardData{Number: "99"}), Server: enc(t, v, clientmodel.CardData{Number: "99"})},
	}}
	m := loadModel(t, New(Prop{Vault: v, Repo: repo, Sync: &fakeSyncer{}}))
	if !strings.Contains(m.View().Content, "99") {
		t.Fatalf("short number not rendered: %q", m.View().Content)
	}
}
