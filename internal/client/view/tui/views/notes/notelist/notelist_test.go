package notelist

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
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

func encNote(t *testing.T, v *vault.Vault, d clientmodel.NoteData) []byte {
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

type fakeRepo struct {
	rows    []repository.NoteRow
	listErr error
	delID   string
	delErr  error
	updErr  error
}

func (f *fakeRepo) List(_ context.Context, _ string, _ int) ([]repository.NoteRow, error) {
	return f.rows, f.listErr
}

func (f *fakeRepo) Update(_ context.Context, _ string, _ []byte) error { return f.updErr }

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.delID = id
	return f.delErr
}

func drive(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	for _, msg := range flatten(cmd) {

		if isTick(msg) {
			continue
		}
		var next tea.Cmd
		m, next = m.Update(msg)
		_ = next
	}
	return m
}

func isTick(msg tea.Msg) bool {
	return strings.Contains(reflect.TypeOf(msg).String(), "TickMsg")
}

func TestDecodeNoteRoundTrip(t *testing.T) {
	v := testVault(t)
	want := clientmodel.NoteData{Text: "secret note", Meta: "tag"}
	row := repository.NoteRow{ID: "n1", Data: encNote(t, v, want), Version: 9}
	got, err := decodeNote(v, row)
	if err != nil {
		t.Fatalf("decodeNote: %v", err)
	}
	if got.ID != "n1" || got.Version != 9 || got.Data != want {
		t.Errorf("decodeNote = %+v", got)
	}
}

func TestDecodeNoteCorrupt(t *testing.T) {
	v := testVault(t)
	if _, err := decodeNote(v, repository.NoteRow{ID: "x", Data: []byte("bad")}); err == nil {
		t.Fatal("expected decrypt error")
	}
}

func TestDecodeNoteBadJSON(t *testing.T) {
	v := testVault(t)
	blob, err := v.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := decodeNote(v, repository.NoteRow{ID: "x", Data: blob}); err == nil {
		t.Fatal("expected json error")
	}
}

func TestSnippet(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "—"},
		{"line1\nline2", "line1 line2"},
		{"short", "short"},
	}
	for _, c := range cases {
		if got := snippet(c.in); got != c.want {
			t.Errorf("snippet(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	long := strings.Repeat("a", colText+10)
	got := snippet(long)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("snippet(long) not truncated: %q", got)
	}
	if len([]rune(got)) > colText {
		t.Errorf("snippet(long) too long: %d runes", len([]rune(got)))
	}
}

func TestRenderItem(t *testing.T) {
	out := renderItem(clientmodel.Note{Data: clientmodel.NoteData{Text: "hi", Meta: "m"}})
	if !strings.Contains(out, "hi") || !strings.Contains(out, "m") {
		t.Errorf("renderItem = %q", out)
	}

	out = renderItem(clientmodel.Note{Data: clientmodel.NoteData{Text: "hi"}})
	if !strings.Contains(out, "—") {
		t.Errorf("renderItem empty meta = %q", out)
	}
}

func TestRenderDetail(t *testing.T) {
	out := renderDetail(clientmodel.Note{Data: clientmodel.NoteData{Text: "body", Meta: "meta"}})
	if !strings.Contains(out, "body") || !strings.Contains(out, "meta") {
		t.Errorf("renderDetail = %q", out)
	}

	out = renderDetail(clientmodel.Note{Data: clientmodel.NoteData{}})
	if !strings.Contains(out, "—") {
		t.Errorf("renderDetail empty = %q", out)
	}
}

func loadedModel(t *testing.T, repo Repo, v *vault.Vault) tea.Model {
	t.Helper()
	m := New(Prop{Vault: v, Repo: repo})
	return drive(t, m, m.Init())
}

func TestNewAndLoad(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.NoteRow{
		{ID: "n1", Data: encNote(t, v, clientmodel.NoteData{Text: "first", Meta: "a"}), Version: 1},
		{ID: "n2", Data: encNote(t, v, clientmodel.NoteData{Text: "second", Meta: "b"}), Version: 1},
	}}
	m := loadedModel(t, repo, v)
	content := m.View().Content
	if !strings.Contains(content, "first") || !strings.Contains(content, "second") {
		t.Errorf("list view missing items: %q", content)
	}
}

func TestLoadError(t *testing.T) {
	v := testVault(t)
	m := loadedModel(t, &fakeRepo{listErr: errors.New("db down")}, v)
	if !strings.Contains(m.View().Content, "Could not load") {
		t.Errorf("view missing load error: %q", m.View().Content)
	}
}

func TestEmptyList(t *testing.T) {
	v := testVault(t)
	m := loadedModel(t, &fakeRepo{rows: nil}, v)
	if !strings.Contains(strings.ToLower(m.View().Content), "no notes") {
		t.Errorf("view missing empty state: %q", m.View().Content)
	}
}

func TestNavigateAndReveal(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.NoteRow{
		{ID: "n1", Data: encNote(t, v, clientmodel.NoteData{Text: "first", Meta: "a"})},
		{ID: "n2", Data: encNote(t, v, clientmodel.NoteData{Text: "second", Meta: "b"})},
	}}
	m := loadedModel(t, repo, v)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.View().Content, "Text") {
		t.Errorf("expected detail view to contain %q, got %q", "Text", m.View().Content)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
}

func TestDeleteFlow(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.NoteRow{
		{ID: "n1", Data: encNote(t, v, clientmodel.NoteData{Text: "first", Meta: "a"})},
	}}
	m := loadedModel(t, repo, v)

	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !strings.Contains(m.View().Content, "delete selected") {
		t.Errorf("d did not start confirm: %q", m.View().Content)
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	flatten(cmd)
	if repo.delID != "n1" {
		t.Errorf("Delete id = %q, want n1", repo.delID)
	}
}

func TestDeleteCancel(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.NoteRow{
		{ID: "n1", Data: encNote(t, v, clientmodel.NoteData{Text: "first", Meta: "a"})},
	}}
	m := loadedModel(t, repo, v)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if strings.Contains(m.View().Content, "delete selected") {
		t.Error("n did not cancel confirm")
	}
	if repo.delID != "" {
		t.Error("Delete called after cancel")
	}
}

func TestEditPushesModel(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.NoteRow{
		{ID: "n1", Data: encNote(t, v, clientmodel.NoteData{Text: "first", Meta: "a"})},
	}}
	m := loadedModel(t, repo, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if cmd == nil {
		t.Fatal("e produced no cmd")
	}

	found := false
	for _, msg := range flatten(cmd) {
		if strings.Contains(reflect.TypeOf(msg).String(), "PushModel") {
			found = true
		}
	}
	if !found {
		t.Fatal("e did not push an edit model")
	}
}

func TestEscBacks(t *testing.T) {
	v := testVault(t)
	m := loadedModel(t, &fakeRepo{rows: nil}, v)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	found := false
	for _, msg := range flatten(cmd) {
		if strings.Contains(reflect.TypeOf(msg).String(), "BackMsg") {
			found = true
		}
	}
	if !found {
		t.Fatal("esc did not produce BackMsg")
	}
}

func TestAddAndConflictKeys(t *testing.T) {
	v := testVault(t)
	m := loadedModel(t, &fakeRepo{rows: nil}, v)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if !pushesScreen(flatten(cmd)) {
		t.Error("a did not push a screen")
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if !pushesScreen(flatten(cmd)) {
		t.Error("c did not push a screen")
	}
}

func pushesScreen(msgs []tea.Msg) bool {
	for _, msg := range msgs {
		if strings.Contains(reflect.TypeOf(msg).String(), "PushMsg") {
			return true
		}
	}
	return false
}
