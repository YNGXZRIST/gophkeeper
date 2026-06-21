package picker

import (
	"strings"
	"testing"

	"gophkeeper/internal/client/view/tui/components/nav"

	tea "charm.land/bubbletea/v2"
)

func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

type marker struct{}

func TestInitNil(t *testing.T) {
	m := New("Pick", []Item{{Label: "A"}})
	if m.Init() != nil {
		t.Error("Init should return nil")
	}
}

func TestView(t *testing.T) {
	m := New("Pick Title", []Item{{Label: "Alpha"}, {Label: "Beta"}})
	out := m.View().Content
	for _, part := range []string{"Pick Title", "Alpha", "Beta"} {
		if !strings.Contains(out, part) {
			t.Errorf("View missing %q", part)
		}
	}
}

func TestSelectedRunsAction(t *testing.T) {
	m := New("Pick", []Item{{Label: "A", Action: func() tea.Msg { return marker{} }}})
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("expected action cmd")
	}
	if _, ok := cmd().(marker); !ok {
		t.Error("action cmd did not return marker msg")
	}
}

func TestSelectedNilAction(t *testing.T) {
	m := New("Pick", []Item{{Label: "A"}})
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd != nil {
		t.Error("nil action should yield nil cmd")
	}
}

func TestBack(t *testing.T) {
	m := New("Pick", []Item{{Label: "A"}})
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("expected back cmd")
	}
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Error("esc did not produce BackMsg")
	}
}

func TestQuit(t *testing.T) {
	m := New("Pick", []Item{{Label: "A"}})
	_, cmd := m.Update(runeKey('q'))
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q did not produce QuitMsg, got %T", cmd())
	}
}

func TestNavigateThenSelect(t *testing.T) {
	m := New("Pick", []Item{
		{Label: "A", Action: func() tea.Msg { return "a" }},
		{Label: "B", Action: func() tea.Msg { return "b" }},
	})
	m, _ = m.Update(specialKey(tea.KeyDown))
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil || cmd() != "b" {
		t.Error("expected second item action after moving cursor down")
	}
}
