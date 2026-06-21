package passwordadd

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

type fakeRepo struct {
	data []byte
	err  error
}

func (f *fakeRepo) Create(_ context.Context, data []byte) (repository.PasswordRow, error) {
	f.data = data
	return repository.PasswordRow{ID: "new"}, f.err
}

func keyRune(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func typeStr(m tea.Model, s string) tea.Model {
	for _, r := range s {
		m, _ = m.Update(keyRune(r))
	}
	return m
}

func down(m tea.Model) tea.Model {
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	return m
}

func TestNewAndSubmitCreate(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{}
	m := New(Prop{Vault: vlt, Repo: repo})
	if m.Init() == nil {
		t.Fatal("Init nil")
	}

	m = typeStr(m, "bob")
	m = down(m)
	m = typeStr(m, "pw123")
	m = down(m)
	m = typeStr(m, "meta")
	m = down(m)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submit nil cmd")
	}
	if cmd() == nil {
		t.Fatal("save msg nil")
	}
	if repo.data == nil {
		t.Fatal("Create not called")
	}
	raw, err := vlt.Decrypt(repo.data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	var got clientmodel.PasswordData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := clientmodel.PasswordData{Login: "bob", Password: "pw123", Meta: "meta"}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubmitCreateError(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{err: errors.New("boom")}
	m := New(Prop{Vault: vlt, Repo: repo})
	m = typeStr(m, "a")
	m = down(m)
	m = typeStr(m, "b")
	m = down(m)
	m = typeStr(m, "c")
	m = down(m)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd() == nil {
		t.Fatal("expected saved msg")
	}
	if repo.data == nil {
		t.Fatal("Create not called")
	}
}

func TestViewTitle(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}})
	if !strings.Contains(m.View().Content, "Password") {
		t.Fatalf("view=%q", m.View().Content)
	}
}
