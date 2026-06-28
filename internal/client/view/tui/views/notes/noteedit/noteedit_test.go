package noteedit

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
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
	id      string
	updated []byte
	err     error
}

func (f *fakeRepo) Update(_ context.Context, id string, data []byte) error {
	f.id = id
	f.updated = data
	return f.err
}

func newNote() clientmodel.Note {
	return clientmodel.Note{ID: "note-1", Data: clientmodel.NoteData{Text: "old", Meta: "m"}, Version: 3}
}

func driveSubmit(t *testing.T, m tea.Model) tea.Cmd {
	t.Helper()
	var cmd tea.Cmd
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return cmd
}

func TestNewPrefillsValues(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Note: newNote()})
	if !strings.Contains(m.View().Content, "Edit note") {
		t.Errorf("view missing title: %q", m.View().Content)
	}
}

func TestSubmitUpdates(t *testing.T) {
	repo := &fakeRepo{}
	m := New(Prop{Vault: testVault(t), Repo: repo, Note: newNote()})
	cmd := driveSubmit(t, m)
	if cmd == nil {
		t.Fatal("submit produced no cmd")
	}
	flatten(cmd)
	if repo.id != "note-1" {
		t.Errorf("Update id = %q, want note-1", repo.id)
	}
	if len(repo.updated) == 0 {
		t.Fatal("Update not called with ciphertext")
	}
}

func TestSubmitUpdateError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("update failed")}
	m := New(Prop{Vault: testVault(t), Repo: repo, Note: newNote()})
	flatten(driveSubmit(t, m))
	if repo.id != "note-1" {
		t.Fatal("Update not called")
	}
}

func TestEscBacks(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Note: newNote()})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	found := false
	for _, msg := range flatten(cmd) {
		if _, ok := msg.(nav.BackMsg); ok {
			found = true
		}
	}
	if !found {
		t.Fatal("esc did not produce BackMsg")
	}
}
