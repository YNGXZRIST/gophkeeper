package menu

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func rune_(r rune) tea.KeyPressMsg   { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func special(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func TestUpdateNonKeyMsg(t *testing.T) {
	m := New("Menu", []string{"One", "Two"})
	m, act := m.Update(struct{}{})
	if act != None {
		t.Errorf("act = %v, want None", act)
	}
	if m.Cursor() != 0 {
		t.Errorf("cursor = %d, want 0", m.Cursor())
	}
}

func TestUpdateDownWrap(t *testing.T) {
	m := New("Menu", []string{"One", "Two"})
	m, _ = m.Update(special(tea.KeyDown))
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor())
	}
	m, _ = m.Update(special(tea.KeyDown))
	if m.Cursor() != 0 {
		t.Errorf("cursor after wrap = %d, want 0", m.Cursor())
	}
}

func TestUpdateUpWrap(t *testing.T) {
	m := New("Menu", []string{"One", "Two"})
	m, _ = m.Update(special(tea.KeyUp))
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1 (wrap to last)", m.Cursor())
	}
}

func TestUpdateActions(t *testing.T) {
	m := New("Menu", []string{"One"})
	if _, act := m.Update(special(tea.KeyEnter)); act != Selected {
		t.Errorf("enter act = %v, want Selected", act)
	}
	if _, act := m.Update(special(tea.KeyEscape)); act != Back {
		t.Errorf("esc act = %v, want Back", act)
	}
	if _, act := m.Update(rune_('q')); act != Quit {
		t.Errorf("q act = %v, want Quit", act)
	}
}

func TestView(t *testing.T) {
	m := New("Menu Title", []string{"Alpha", "Beta"})
	out := m.View()
	for _, part := range []string{"Menu Title", "Alpha", "Beta"} {
		if !strings.Contains(out, part) {
			t.Errorf("View missing %q", part)
		}
	}

	if !strings.Contains(out, ">") {
		t.Error("View missing cursor marker")
	}
}
