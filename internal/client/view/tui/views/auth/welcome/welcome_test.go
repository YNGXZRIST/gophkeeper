package welcome

import (
	"testing"

	"gophkeeper/internal/client/view/tui/components/nav"

	tea "charm.land/bubbletea/v2"
)

func TestNewInitView(t *testing.T) {
	m := New()
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should return nil cmd")
	}
	if m.View().Content == "" {
		t.Fatal("View content is empty")
	}
}

func TestUpdateSelectSignIn(t *testing.T) {
	m := New()
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected nav cmd")
	}
	push, ok := cmd().(nav.PushMsg)
	if !ok || push.ID != nav.Login {
		t.Fatalf("expected Push(Login), got %#v", cmd())
	}
	_ = m
}

func TestUpdateSelectSignUp(t *testing.T) {
	m := New()

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected nav cmd")
	}
	push, ok := cmd().(nav.PushMsg)
	if !ok || push.ID != nav.Register {
		t.Fatalf("expected Push(Register), got %#v", cmd())
	}
}

func TestUpdateBack(t *testing.T) {
	m := New()
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected back cmd")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %#v", cmd())
	}
}

func TestUpdateQuit(t *testing.T) {
	m := New()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %#v", cmd())
	}
}

func TestUpdateNoAction(t *testing.T) {
	m := New()

	_, cmd := m.Update(struct{}{})
	if cmd != nil {
		t.Fatalf("expected nil cmd for unhandled msg, got %#v", cmd())
	}
}
