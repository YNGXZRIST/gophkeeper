package unlock

import (
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/crypto"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func sessionFor(t *testing.T, password string) *auth.Session {
	t.Helper()
	salt, err := crypto.GenerateBytes(16)
	if err != nil {
		t.Fatalf("salt: %v", err)
	}
	dek, err := crypto.GenerateBytes(32)
	if err != nil {
		t.Fatalf("dek: %v", err)
	}
	enc, err := crypto.NewEncryptor(crypto.DeriveKey(password, salt))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	wrapped, err := enc.Encrypt(dek)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	return auth.NewSession("alice", auth.Token{}, auth.Token{}, salt, wrapped)
}

func typePassword(t *testing.T, m tea.Model, pw string) tea.Model {
	t.Helper()
	for _, r := range pw {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func submit(t *testing.T, m tea.Model) (tea.Model, tea.Cmd) {
	t.Helper()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	return m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
}

func TestNewInitView(t *testing.T) {
	m := New(vault.New(), sessionFor(t, "pw"))
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	if !strings.Contains(m.View().Content, "Unlock") {
		t.Fatal("View missing title")
	}
}

func TestUpdateCtrlC(t *testing.T) {
	m := New(vault.New(), sessionFor(t, "pw"))
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %#v", cmd())
	}
}

func TestUpdateEscLogout(t *testing.T) {
	m := New(vault.New(), sessionFor(t, "pw"))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected logout cmd")
	}
	if _, ok := cmd().(nav.LogoutMsg); !ok {
		t.Fatalf("expected LogoutMsg, got %#v", cmd())
	}
}

func TestUpdateSubmitSuccess(t *testing.T) {
	v := vault.New()
	m := New(v, sessionFor(t, "correct"))
	m = typePassword(t, m, "correct")
	m, cmd := submit(t, m)
	if cmd == nil {
		t.Fatal("expected reset cmd")
	}
	reset, ok := cmd().(nav.ResetMsg)
	if !ok || reset.ID != nav.Home {
		t.Fatalf("expected Reset(Home), got %#v", cmd())
	}
	if v.Locked() {
		t.Fatal("vault should be unlocked after success")
	}
}

func TestUpdateSubmitWrongPassword(t *testing.T) {
	v := vault.New()
	mm := New(v, sessionFor(t, "correct"))
	mm = typePassword(t, mm, "wrong")
	mm, cmd := submit(t, mm)
	if cmd != nil {
		t.Fatalf("expected nil cmd on error, got %#v", cmd())
	}
	if !strings.Contains(mm.View().Content, "Wrong password") {
		t.Fatalf("expected wrong-password message, got %q", mm.View().Content)
	}
}

func TestUpdateSubmitCrash(t *testing.T) {
	v := vault.New()
	mm := New(v, sessionFor(t, "correct"))
	mm = typePassword(t, mm, "nope")
	mm, _ = submit(t, mm)
	if !strings.Contains(mm.View().Content, "Wrong password") {
		t.Fatalf("expected error display, got %q", mm.View().Content)
	}
}

func TestUpdateFormPassthrough(t *testing.T) {
	m := New(vault.New(), sessionFor(t, "pw"))
	m, _ = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if strings.Contains(m.View().Content, "Wrong password") {
		t.Fatal("unexpected error message before submit")
	}
}
