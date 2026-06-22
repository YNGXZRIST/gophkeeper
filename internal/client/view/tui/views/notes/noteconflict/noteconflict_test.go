package noteconflict

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

func enc(t *testing.T, v *vault.Vault, d clientmodel.NoteData) []byte {
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

func flatten(msg tea.Msg) []tea.Msg {
	if msg == nil {
		return nil
	}
	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			if c != nil {
				out = append(out, flatten(c())...)
			}
		}
		return out
	case tea.Cmd:
		if m == nil {
			return nil
		}
		return flatten(m())
	}
	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		var out []tea.Msg
		for i := 0; i < rv.Len(); i++ {
			if c, ok := rv.Index(i).Interface().(tea.Cmd); ok && c != nil {
				out = append(out, flatten(c())...)
			}
		}
		return out
	}
	return []tea.Msg{msg}
}

func hasType[T any](msgs []tea.Msg) bool {
	for _, msg := range msgs {
		if _, ok := msg.(T); ok {
			return true
		}
	}
	return false
}

type fakeRepo struct {
	rows        []repository.ConflictRow
	listErr     error
	keepMineErr error
	takeServErr error
	keptID      string
	tookID      string
}

func (f *fakeRepo) ListConflicts(context.Context) ([]repository.ConflictRow, error) {
	return f.rows, f.listErr
}

func (f *fakeRepo) ResolveKeepMine(_ context.Context, id string) error {
	f.keptID = id
	return f.keepMineErr
}

func (f *fakeRepo) ResolveTakeServer(_ context.Context, id string) error {
	f.tookID = id
	return f.takeServErr
}

type fakeSync struct {
	err    error
	called bool
}

func (f *fakeSync) SyncAll(context.Context) error { f.called = true; return f.err }

func rows(t *testing.T, v *vault.Vault) []repository.ConflictRow {
	return []repository.ConflictRow{
		{ID: "c1", Local: enc(t, v, clientmodel.NoteData{Text: "mine-1"}), Server: enc(t, v, clientmodel.NoteData{Text: "srv-1"}), ServerVersion: 2},
		{ID: "c2", Local: enc(t, v, clientmodel.NoteData{Text: "mine-2"}), Server: enc(t, v, clientmodel.NoteData{Text: "srv-2"}), ServerVersion: 5},
	}
}

func loaded(t *testing.T, repo Repo, sync Syncer, v *vault.Vault) tea.Model {
	t.Helper()
	m := New(Prop{Vault: v, Repo: repo, Sync: sync})

	syncMsgs := flatten(m.Init())
	if !hasType[syncedMsg](syncMsgs) {
		t.Fatal("Init did not produce syncedMsg")
	}
	var cmd tea.Cmd
	for _, msg := range syncMsgs {
		if sm, ok := msg.(syncedMsg); ok {
			m, cmd = m.Update(sm)
		}
	}
	for _, msg := range flatten(cmd) {
		if lm, ok := msg.(loadedMsg); ok {
			m, _ = m.Update(lm)
		}
	}
	return m
}

func TestInitSyncSuccessLoads(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v)}
	sync := &fakeSync{}
	m := loaded(t, repo, sync, v)
	if !sync.called {
		t.Fatal("SyncAll not called")
	}
	content := m.View().Content
	if !strings.Contains(content, "mine-1") {
		t.Errorf("view missing local note: %q", content)
	}
	if !strings.Contains(content, "MINE") || !strings.Contains(content, "SERVER") {
		t.Errorf("view missing MINE/SERVER labels: %q", content)
	}
	if !strings.Contains(content, "srv-1") {
		t.Errorf("view missing server note: %q", content)
	}
}

func TestInitSyncError(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v)}
	m := New(Prop{Vault: v, Repo: repo, Sync: &fakeSync{err: errors.New("net down")}})

	mm, _ := m.Update(syncedMsg{err: errors.New("net down")})

	mm, _ = mm.Update(loadedMsg{items: nil})
	if !strings.Contains(mm.View().Content, "Sync error") {
		t.Errorf("view missing sync error: %q", mm.View().Content)
	}
}

func TestNilSyncer(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v)}
	m := loaded(t, repo, nil, v)
	if !strings.Contains(m.View().Content, "mine-1") {
		t.Errorf("nil syncer did not load: %q", m.View().Content)
	}
}

func TestLoadError(t *testing.T) {
	v := testVault(t)
	m := New(Prop{Vault: v, Repo: &fakeRepo{listErr: errors.New("db gone")}, Sync: &fakeSync{}})
	mm, cmd := m.Update(syncedMsg{})
	for _, msg := range flatten(cmd) {
		if lm, ok := msg.(loadedMsg); ok {
			mm, _ = mm.Update(lm)
		}
	}
	if !strings.Contains(mm.View().Content, "Load error") {
		t.Errorf("view missing load error: %q", mm.View().Content)
	}
}

func TestEmptyConflicts(t *testing.T) {
	v := testVault(t)
	m := loaded(t, &fakeRepo{rows: nil}, &fakeSync{}, v)
	if !strings.Contains(m.View().Content, "No conflicts") {
		t.Errorf("view missing 'No conflicts': %q", m.View().Content)
	}
}

func TestNavigationKeys(t *testing.T) {
	v := testVault(t)
	m := loaded(t, &fakeRepo{rows: rows(t, v)}, &fakeSync{}, v)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !strings.Contains(m.View().Content, "srv-2") {
		t.Errorf("down did not move selection: %q", m.View().Content)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if !strings.Contains(m.View().Content, "srv-1") {
		t.Errorf("up did not move selection: %q", m.View().Content)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
}

func TestKeepMine(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v)}
	sync := &fakeSync{}
	m := loaded(t, repo, sync, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	msgs := flatten(cmd)
	if repo.keptID != "c1" {
		t.Errorf("ResolveKeepMine id = %q, want c1", repo.keptID)
	}
	if !hasType[resolvedMsg](msgs) {
		t.Fatalf("m key did not produce resolvedMsg: %#v", msgs)
	}
}

func TestTakeServer(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v)}
	m := loaded(t, repo, &fakeSync{}, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	msgs := flatten(cmd)
	if repo.tookID != "c1" {
		t.Errorf("ResolveTakeServer id = %q, want c1", repo.tookID)
	}
	if !hasType[resolvedMsg](msgs) {
		t.Fatal("s key did not produce resolvedMsg")
	}
}

func TestResolvedMsgReSync(t *testing.T) {
	v := testVault(t)
	sync := &fakeSync{}
	m := loaded(t, &fakeRepo{rows: rows(t, v)}, sync, v)
	mm, cmd := m.Update(resolvedMsg{})

	if !strings.Contains(mm.View().Content, "Syncing") {
		t.Errorf("resolvedMsg did not set loading: %q", mm.View().Content)
	}
	msgs := flatten(cmd)
	if !hasType[nav.SyncNowMsg](msgs) {
		t.Fatal("resolvedMsg did not fire SyncNow")
	}
	if !hasType[syncedMsg](msgs) {
		t.Fatal("resolvedMsg did not re-run sync")
	}
}

func TestResolvedMsgError(t *testing.T) {
	v := testVault(t)
	m := loaded(t, &fakeRepo{rows: rows(t, v)}, &fakeSync{}, v)
	mm, cmd := m.Update(resolvedMsg{err: errors.New("conflict")})
	if cmd != nil {
		t.Error("expected nil cmd on resolve error")
	}
	if !strings.Contains(mm.View().Content, "Resolve error") {
		t.Errorf("view missing resolve error: %q", mm.View().Content)
	}
}

func TestResolveErrorFromRepo(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: rows(t, v), keepMineErr: errors.New("boom")}
	m := loaded(t, repo, &fakeSync{}, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	msgs := flatten(cmd)
	for _, msg := range msgs {
		if rm, ok := msg.(resolvedMsg); ok && rm.err == nil {
			t.Fatal("expected resolvedMsg carrying error")
		}
	}
}

func TestEscBacks(t *testing.T) {
	v := testVault(t)
	m := loaded(t, &fakeRepo{rows: rows(t, v)}, &fakeSync{}, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !hasType[nav.BackMsg](flatten(cmd)) {
		t.Fatal("esc did not produce BackMsg")
	}
}

func TestLoadingView(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Sync: &fakeSync{}})

	if !strings.Contains(m.View().Content, "Syncing") {
		t.Errorf("initial view should be loading: %q", m.View().Content)
	}
}

func TestKeysOnEmptyNoSelection(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: nil}
	m := loaded(t, repo, &fakeSync{}, v)

	m, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	_, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if repo.keptID != "" || repo.tookID != "" {
		t.Error("resolve called with no selection")
	}
}

func TestLineEmptyText(t *testing.T) {
	if got := line(clientmodel.NoteData{}); got != "—" {
		t.Errorf("line(empty) = %q, want —", got)
	}
	if got := line(clientmodel.NoteData{Text: "a\nb"}); got != "a b" {
		t.Errorf("line newline replace = %q", got)
	}
}

func TestDecodeCorrupt(t *testing.T) {
	v := testVault(t)

	if got := decode(v, []byte("garbage")); got != (clientmodel.NoteData{}) {
		t.Errorf("decode(corrupt) = %+v, want empty", got)
	}

	blob, err := v.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got := decode(v, blob); got != (clientmodel.NoteData{}) {
		t.Errorf("decode(bad json) = %+v, want empty", got)
	}
}

func TestDecodeRoundTrip(t *testing.T) {
	v := testVault(t)
	want := clientmodel.NoteData{Text: "hi", Meta: "m"}
	if got := decode(v, enc(t, v, want)); got != want {
		t.Errorf("decode = %+v, want %+v", got, want)
	}
}
