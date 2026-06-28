package passwordedit

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
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
	id   string
	data []byte
	err  error
}

func (f *fakeRepo) Update(_ context.Context, id string, data []byte) error {
	f.id = id
	f.data = data
	return f.err
}

func keyRune(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func down(m tea.Model) tea.Model {
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	return m
}

func TestNewPrefillsAndSubmitUpdate(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{}
	pw := clientmodel.Password{
		ID:   "pw-7",
		Data: clientmodel.PasswordData{Login: "u", Password: "p", Meta: "m"},
	}
	m := New(Prop{Vault: vlt, Repo: repo, Password: pw})
	if m.Init() == nil {
		t.Fatal("Init nil")
	}
	if !strings.Contains(m.View().Content, "Edit password") {
		t.Fatalf("title view=%q", m.View().Content)
	}

	m = down(m)
	m = down(m)
	m = down(m)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submit nil cmd")
	}
	if cmd() == nil {
		t.Fatal("saved msg nil")
	}
	if repo.id != "pw-7" {
		t.Fatalf("update id=%q", repo.id)
	}
	raw, err := vlt.Decrypt(repo.data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	var got clientmodel.PasswordData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != pw.Data {
		t.Fatalf("got %+v want %+v", got, pw.Data)
	}
}

func TestSubmitUpdateError(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{err: errors.New("fail")}
	pw := clientmodel.Password{ID: "x", Data: clientmodel.PasswordData{Login: "a", Password: "b", Meta: "c"}}
	m := New(Prop{Vault: vlt, Repo: repo, Password: pw})
	m = down(m)
	m = down(m)
	m = down(m)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd() == nil {
		t.Fatal("saved msg nil")
	}
	if repo.data == nil {
		t.Fatal("Update not called")
	}
}
