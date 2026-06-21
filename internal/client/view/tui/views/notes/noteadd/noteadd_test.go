package noteadd

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"reflect"
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
	created []byte
	err     error
}

func (f *fakeRepo) Create(_ context.Context, data []byte) (repository.NoteRow, error) {
	f.created = data
	return repository.NoteRow{ID: "new-id"}, f.err
}

func driveSubmit(t *testing.T, m tea.Model) tea.Cmd {
	t.Helper()
	var cmd tea.Cmd

	m, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return cmd
}

func TestNewSubmitCreates(t *testing.T) {
	repo := &fakeRepo{}
	m := New(Prop{Vault: testVault(t), Repo: repo})
	if m == nil {
		t.Fatal("New returned nil")
	}
	cmd := driveSubmit(t, m)
	if cmd == nil {
		t.Fatal("submit produced no cmd")
	}

	flatten(cmd)
	if len(repo.created) == 0 {
		t.Fatal("Create was not called with ciphertext")
	}
}

func TestNewSubmitError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("create failed")}
	m := New(Prop{Vault: testVault(t), Repo: repo})
	cmd := driveSubmit(t, m)
	msgs := flatten(cmd)
	if len(repo.created) == 0 {
		t.Fatal("Create not called")
	}
	_ = msgs
}

func TestNewEscBacks(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}})
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

func TestNewRendersTitle(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}})
	if m.View().Content == "" {
		t.Fatal("view content is empty")
	}
	_ = clientmodel.NoteData{}
}
