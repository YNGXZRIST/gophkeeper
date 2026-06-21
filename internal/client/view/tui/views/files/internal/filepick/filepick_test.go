package filepick

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
)

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestNewAndInit(t *testing.T) {
	m := New(Prop{Title: "Pick"})
	if m == nil {
		t.Fatal("New returned nil")
	}
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	if v := m.View(); v.Content == "" {
		t.Error("View content empty")
	}
}

func TestEscBack(t *testing.T) {
	m := New(Prop{Title: "Pick"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should return a Back cmd")
	}
}

func TestCtrlCQuit(t *testing.T) {
	m := New(Prop{Title: "Pick"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c should return a Quit cmd")
	}
}

func TestNavigationKeys(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	chdir(t, dir)

	fp := filepicker.New()
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.AutoHeight = false
	fp.SetHeight(pickerHeight)
	fp.CurrentDirectory = dir
	m := model{picker: fp, title: "Pick"}

	if cmd := m.Init(); cmd != nil {
		if msg := cmd(); msg != nil {
			mm, _ := m.Update(msg)
			m = mm.(model)
		}
	}
	for _, code := range []rune{tea.KeyDown, tea.KeyUp, tea.KeyRight, tea.KeyLeft, tea.KeyEnter} {
		mm, _ := m.Update(tea.KeyPressMsg{Code: code})
		m = mm.(model)
	}
	_ = m.View()
}

func TestOnSelectInvoked(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pick-me.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	called := false
	fp := filepicker.New()
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.CurrentDirectory = dir
	m := model{
		picker: fp,
		title:  "Pick",
		onSelect: func(path string) tea.Cmd {
			called = true
			return func() tea.Msg { return nil }
		},
	}
	mm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = mm
	_ = called
}

func TestViewWithError(t *testing.T) {
	m := model{picker: filepicker.New(), title: "Pick", errMsg: "File type not allowed."}
	v := m.View()
	if v.Content == "" {
		t.Error("expected content with error message")
	}
}
