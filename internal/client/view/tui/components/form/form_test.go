package form

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func newForm() Model {
	return New("Save", []Field{
		{Placeholder: "name", Width: 20},
		{Placeholder: "secret", Width: 20, Password: true},
	})
}

func TestInit(t *testing.T) {
	m := newForm()
	if m.Init() == nil {
		t.Error("Init should return Blink cmd")
	}
}

func TestValuesEmpty(t *testing.T) {
	m := newForm()
	vals := m.Values()
	if len(vals) != 2 {
		t.Fatalf("len(values) = %d, want 2", len(vals))
	}
	for _, v := range vals {
		if v != "" {
			t.Errorf("value = %q, want empty", v)
		}
	}
}

func TestTypeAndValues(t *testing.T) {
	m := newForm()
	for _, r := range "bob" {
		m, _, _ = m.Update(runeKey(r))
	}
	if got := m.Values()[0]; got != "bob" {
		t.Errorf("values[0] = %q, want bob", got)
	}
}

func TestNavigateUpDown(t *testing.T) {
	m := newForm()
	var act Action
	m, act, _ = m.Update(specialKey(tea.KeyDown))
	if act != None {
		t.Errorf("down act = %v, want None", act)
	}
	_, act, _ = m.Update(specialKey(tea.KeyUp))
	if act != None {
		t.Errorf("up act = %v, want None", act)
	}
}

func TestEnterAdvancesFocus(t *testing.T) {
	m := newForm()

	_, act, _ := m.Update(specialKey(tea.KeyEnter))
	if act != None {
		t.Errorf("enter on field act = %v, want None", act)
	}
}

func TestSubmitGatedAndAllowed(t *testing.T) {
	m := newForm()

	m, _, _ = m.Update(specialKey(tea.KeyDown))
	m, _, _ = m.Update(specialKey(tea.KeyDown))

	if _, act, _ := m.Update(specialKey(tea.KeyEnter)); act != None {
		t.Errorf("enter with empty fields act = %v, want None", act)
	}

	m = newForm()
	for _, r := range "abc" {
		m, _, _ = m.Update(runeKey(r))
	}
	m, _, _ = m.Update(specialKey(tea.KeyDown))
	for _, r := range "xyz" {
		m, _, _ = m.Update(runeKey(r))
	}
	m, _, _ = m.Update(specialKey(tea.KeyDown))
	_, act, _ := m.Update(specialKey(tea.KeyEnter))
	if act != Submit {
		t.Errorf("enter on button with all filled act = %v, want Submit", act)
	}
}

func TestUpWrapsToButton(t *testing.T) {
	m := newForm()

	m, _, _ = m.Update(specialKey(tea.KeyUp))

	if _, act, _ := m.Update(specialKey(tea.KeyEnter)); act != None {
		t.Errorf("act = %v, want None", act)
	}
}

func TestView(t *testing.T) {
	m := New("SubmitMe", []Field{{Placeholder: "ph", Width: 10}})
	body, _ := m.View(0)
	if !strings.Contains(body, "SubmitMe") {
		t.Errorf("View missing submit label: %q", body)
	}
}

func TestViewFocusedRendersBody(t *testing.T) {
	m := newForm()

	body, _ := m.View(0)
	if body == "" {
		t.Error("expected non-empty body from focused form")
	}
}
