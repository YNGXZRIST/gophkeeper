package passform

import (
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
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

func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func TestInitReturnsCmd(t *testing.T) {
	m := New(testVault(t), "Password", clientmodel.PasswordData{}, func([]byte) error { return nil })
	if m.Init() == nil {
		t.Fatal("Init returned nil cmd")
	}
}

func TestEscReturnsBack(t *testing.T) {
	m := New(testVault(t), "Password", clientmodel.PasswordData{}, func([]byte) error { return nil })
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc: nil cmd")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("esc: expected BackMsg, got %T", cmd())
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := New(testVault(t), "Password", clientmodel.PasswordData{}, func([]byte) error { return nil })
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl, Text: ""})
	if cmd == nil {
		t.Fatal("ctrl+c: nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c: expected QuitMsg, got %T", cmd())
	}
}

func fillForm(t *testing.T, m Model, login, pw, meta string) Model {
	t.Helper()
	for _, r := range login {
		mm, _ := m.Update(keyRune(r))
		m = mm.(Model)
	}
	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(Model)
	for _, r := range pw {
		mm, _ := m.Update(keyRune(r))
		m = mm.(Model)
	}
	mm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(Model)
	for _, r := range meta {
		mm, _ := m.Update(keyRune(r))
		m = mm.(Model)
	}
	return m
}

func TestSubmitSuccess(t *testing.T) {
	vlt := testVault(t)
	var saved []byte
	m := New(vlt, "Password", clientmodel.PasswordData{}, func(ct []byte) error {
		saved = ct
		return nil
	})
	m = fillForm(t, m, "alice", "secret", "note")

	if m.GetLogin() != "alice" || m.GetPassword() != "secret" || m.GetMeta() != "note" {
		t.Fatalf("getters: %q %q %q", m.GetLogin(), m.GetPassword(), m.GetMeta())
	}

	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(Model)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submit: nil cmd")
	}
	msg := cmd()
	sv, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("submit: expected savedMsg, got %T", msg)
	}
	if sv.err != nil {
		t.Fatalf("submit err: %v", sv.err)
	}
	if saved == nil {
		t.Fatal("save func not called")
	}

	raw, err := vlt.Decrypt(saved)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	var got clientmodel.PasswordData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := clientmodel.PasswordData{Login: "alice", Password: "secret", Meta: "note"}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestSubmitSaveError(t *testing.T) {
	m := New(testVault(t), "Password", clientmodel.PasswordData{}, func([]byte) error {
		return errors.New("boom")
	})
	m = fillForm(t, m, "a", "b", "c")
	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = mm.(Model)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd()
	sv := msg.(savedMsg)
	if sv.err == nil {
		t.Fatal("expected error")
	}

	mm, cmd2 := m.Update(savedMsg{err: errors.New("boom")})
	m = mm.(Model)
	if cmd2 != nil {
		t.Fatalf("error path should return nil cmd, got %v", cmd2)
	}
	if !strings.Contains(m.View().Content, "Save failed") {
		t.Fatalf("view should show save failed: %q", m.View().Content)
	}
}

func TestSavedMsgSuccessTriggersNav(t *testing.T) {
	m := New(testVault(t), "Password", clientmodel.PasswordData{}, func([]byte) error { return nil })
	_, cmd := m.Update(savedMsg{})
	if cmd == nil {
		t.Fatal("savedMsg success: nil cmd")
	}

	if cmd() == nil {
		t.Fatal("sequence produced nil")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := New(testVault(t), "My Title", clientmodel.PasswordData{Login: "x"}, func([]byte) error { return nil })
	if !strings.Contains(m.View().Content, "My Title") {
		t.Fatalf("view missing title: %q", m.View().Content)
	}
}
