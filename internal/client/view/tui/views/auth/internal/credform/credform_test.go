package credform

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func keyPress(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

func TestNewAndInit(t *testing.T) {
	m := New()
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
}

func TestView(t *testing.T) {
	m := New()
	body, _ := m.View()
	if body == "" {
		t.Fatal("View body is empty")
	}
}

func TestUpdateTypeAndNavigate(t *testing.T) {
	m := New()

	var cmd tea.Cmd
	m, cmd = m.Update(keyPress('a', "a"))
	_ = cmd
	if got := m.form.Values()[0]; got != "a" {
		t.Fatalf("login value = %q, want a", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(keyPress('b', "b"))
	if got := m.form.Values()[1]; got != "b" {
		t.Fatalf("password value = %q, want b", got)
	}
}

func TestUpdateSubmit(t *testing.T) {
	m := New()

	m, _ = m.Update(keyPress('u', "u"))
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(keyPress('p', "p"))

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd")
	}
	msg := cmd()
	sub, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if sub.Login != "u" || sub.Password != "p" {
		t.Fatalf("SubmitMsg = %+v, want {u p}", sub)
	}
}
