package passwordconflict

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
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

func enc(t *testing.T, v *vault.Vault, d clientmodel.PasswordData) []byte {
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
	rows    []repository.ConflictRow
	listErr error
	keepErr error
	takeErr error
	keptIDs []string
	tookIDs []string
}

func (f *fakeRepo) ListConflicts(context.Context) ([]repository.ConflictRow, error) {
	return f.rows, f.listErr
}
func (f *fakeRepo) ResolveKeepMine(_ context.Context, id string) error {
	f.keptIDs = append(f.keptIDs, id)
	return f.keepErr
}
func (f *fakeRepo) ResolveTakeServer(_ context.Context, id string) error {
	f.tookIDs = append(f.tookIDs, id)
	return f.takeErr
}

type fakeSyncer struct {
	err    error
	called int
}

func (s *fakeSyncer) SyncAll(context.Context) error {
	s.called++
	return s.err
}

func keyRune(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func newModel(t *testing.T, repo Repo, sync Syncer, vlt *vault.Vault) model {
	t.Helper()
	return New(Prop{Vault: vlt, Repo: repo, Sync: sync}).(model)
}

func drive(t *testing.T, m model) model {
	t.Helper()
	initCmd := m.Init()
	if initCmd == nil {
		t.Fatal("Init nil cmd")
	}
	synced := initCmd()
	mm, loadCmd := m.Update(synced)
	m = mm.(model)
	if loadCmd == nil {
		t.Fatal("load cmd nil after synced")
	}
	loaded := loadCmd()
	mm, _ = m.Update(loaded)
	return mm.(model)
}

func TestDriveLoadsItems(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "p1", Local: enc(t, vlt, clientmodel.PasswordData{Login: "mine", Password: "aaa"}),
			Server: enc(t, vlt, clientmodel.PasswordData{Login: "srv", Password: "bb"}), ServerVersion: 3},
	}}
	sync := &fakeSyncer{}
	m := drive(t, newModel(t, repo, sync, vlt))
	if m.loading {
		t.Fatal("still loading")
	}
	if len(m.items) != 1 {
		t.Fatalf("items = %d", len(m.items))
	}
	content := m.View().Content
	for _, want := range []string{"mine", "MINE", "SERVER"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q: %q", want, content)
		}
	}
	if sync.called != 1 {
		t.Fatalf("sync called %d times", sync.called)
	}
}

func TestNilSyncer(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{}
	m := New(Prop{Vault: vlt, Repo: repo, Sync: nil}).(model)
	cmd := m.Init()
	if _, ok := cmd().(syncedMsg); !ok {
		t.Fatalf("nil syncer: expected syncedMsg")
	}
}

func TestSyncError(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{}
	sync := &fakeSyncer{err: errors.New("netfail")}
	m := newModel(t, repo, sync, vlt)
	synced := m.Init()()
	mm, _ := m.Update(synced)
	m = mm.(model)
	if !strings.Contains(m.status, "Sync error") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestLoadError(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{listErr: errors.New("dbfail")}
	m := newModel(t, repo, &fakeSyncer{}, vlt)
	mm, _ := m.Update(loadedMsg{err: errors.New("dbfail")})
	m = mm.(model)
	if m.loading {
		t.Fatal("loading should be false")
	}
	if !strings.Contains(m.status, "Load error") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestNavigateAndResolveKeepMine(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "p1", Local: enc(t, vlt, clientmodel.PasswordData{Login: "a"}), Server: enc(t, vlt, clientmodel.PasswordData{Login: "b"})},
		{ID: "p2", Local: enc(t, vlt, clientmodel.PasswordData{Login: "c"}), Server: enc(t, vlt, clientmodel.PasswordData{Login: "d"})},
	}}
	m := drive(t, newModel(t, repo, &fakeSyncer{}, vlt))

	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(model)
	if m.selected != 1 {
		t.Fatalf("selected=%d", m.selected)
	}
	mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(model)
	if m.selected != 1 {
		t.Fatalf("selected clamp=%d", m.selected)
	}
	mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = mm.(model)
	mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = mm.(model)
	if m.selected != 0 {
		t.Fatalf("selected after up=%d", m.selected)
	}

	mm, cmd := m.Update(keyRune('m'))
	m = mm.(model)
	if !m.loading {
		t.Fatal("should be loading after resolve")
	}
	msg := cmd()
	rm, ok := msg.(resolvedMsg)
	if !ok {
		t.Fatalf("expected resolvedMsg, got %T", msg)
	}
	if rm.err != nil {
		t.Fatalf("resolve err: %v", rm.err)
	}
	if len(repo.keptIDs) != 1 || repo.keptIDs[0] != "p1" {
		t.Fatalf("keptIDs=%v", repo.keptIDs)
	}
}

func TestResolveTakeServer(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{rows: []repository.ConflictRow{
		{ID: "p1", Local: enc(t, vlt, clientmodel.PasswordData{Login: "a"}), Server: enc(t, vlt, clientmodel.PasswordData{Login: "b"})},
	}}
	m := drive(t, newModel(t, repo, &fakeSyncer{}, vlt))
	mm, cmd := m.Update(keyRune('s'))
	m = mm.(model)
	msg := cmd()
	if _, ok := msg.(resolvedMsg); !ok {
		t.Fatalf("expected resolvedMsg, got %T", msg)
	}
	if len(repo.tookIDs) != 1 || repo.tookIDs[0] != "p1" {
		t.Fatalf("tookIDs=%v", repo.tookIDs)
	}
}

func TestResolvedMsgSuccessReSyncs(t *testing.T) {
	vlt := testVault(t)
	m := newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt)
	mm, cmd := m.Update(resolvedMsg{})
	m = mm.(model)
	if !m.loading {
		t.Fatal("should be loading")
	}
	if cmd == nil {
		t.Fatal("nil cmd")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", cmd())
	}
	var sawSyncNow, sawSynced bool
	for _, c := range batch {
		switch c().(type) {
		case nav.SyncNowMsg:
			sawSyncNow = true
		case syncedMsg:
			sawSynced = true
		}
	}
	if !sawSyncNow || !sawSynced {
		t.Fatalf("batch missing msgs: syncNow=%v synced=%v", sawSyncNow, sawSynced)
	}
}

func TestResolvedMsgError(t *testing.T) {
	vlt := testVault(t)
	m := newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt)
	mm, cmd := m.Update(resolvedMsg{err: errors.New("conflict")})
	m = mm.(model)
	if cmd != nil {
		t.Fatal("error path should be nil cmd")
	}
	if !strings.Contains(m.status, "Resolve error") {
		t.Fatalf("status=%q", m.status)
	}
}

func TestEscBack(t *testing.T) {
	vlt := testVault(t)
	m := drive(t, newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("esc: expected BackMsg, got %T", cmd())
	}
}

func TestViewNoConflicts(t *testing.T) {
	vlt := testVault(t)
	m := drive(t, newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt))
	if !strings.Contains(m.View().Content, "No conflicts") {
		t.Fatalf("view=%q", m.View().Content)
	}
}

func TestViewLoading(t *testing.T) {
	vlt := testVault(t)
	m := newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt)
	if !strings.Contains(m.View().Content, "Syncing") {
		t.Fatalf("loading view=%q", m.View().Content)
	}
}

func TestViewStatusShown(t *testing.T) {
	vlt := testVault(t)
	m := drive(t, newModel(t, &fakeRepo{}, &fakeSyncer{err: errors.New("x")}, vlt))

	m.status = "Sync error: x"
	if !strings.Contains(m.View().Content, "Sync error") {
		t.Fatalf("view=%q", m.View().Content)
	}
}

func TestMOnEmptyNoop(t *testing.T) {
	vlt := testVault(t)
	m := drive(t, newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt))
	mm, cmd := m.Update(keyRune('m'))
	m = mm.(model)
	if cmd != nil {
		t.Fatal("m on empty should noop")
	}
	mm, cmd = m.Update(keyRune('s'))
	if cmd != nil {
		t.Fatal("s on empty should noop")
	}
	_ = mm
}

func TestLineAndMaskAndDecode(t *testing.T) {
	if got := line(clientmodel.PasswordData{}); got != "— · —" {
		t.Fatalf("empty line=%q", got)
	}
	if got := line(clientmodel.PasswordData{Login: "u", Password: "ab"}); got != "u · ••" {
		t.Fatalf("line=%q", got)
	}
	if mask("") != "—" {
		t.Fatal("mask empty")
	}
	if mask("xyz") != "•••" {
		t.Fatal("mask xyz")
	}

	vlt := testVault(t)
	if got := decode(vlt, []byte("garbage")); got != (clientmodel.PasswordData{}) {
		t.Fatalf("decode corrupt = %+v", got)
	}

	bad, err := vlt.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := decode(vlt, bad); got != (clientmodel.PasswordData{}) {
		t.Fatalf("decode bad json = %+v", got)
	}
}

func TestCurrentIDOutOfRange(t *testing.T) {
	m := model{selected: 5}
	if _, ok := m.currentID(); ok {
		t.Fatal("expected false for out of range")
	}
	m = model{selected: -1}
	if _, ok := m.currentID(); ok {
		t.Fatal("expected false for negative")
	}
}

func TestLoadedSelectedReset(t *testing.T) {
	vlt := testVault(t)
	m := newModel(t, &fakeRepo{}, &fakeSyncer{}, vlt)
	m.selected = 9
	mm, _ := m.Update(loadedMsg{items: []item{{id: "a"}}})
	m = mm.(model)
	if m.selected != 0 {
		t.Fatalf("selected not reset: %d", m.selected)
	}
}
