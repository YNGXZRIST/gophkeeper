// Package filepick is a screen for choosing a local file to upload.
package filepick

import (
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	"os"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
)

const hint = "↑/↓ — move · enter — open/select · esc — back"

const pickerHeight = 15

// Prop configures the picker. OnSelect is called with the chosen file's path.
type Prop struct {
	Title    string
	OnSelect func(path string) tea.Cmd
}

type model struct {
	picker   filepicker.Model
	title    string
	onSelect func(path string) tea.Cmd
	errMsg   string
}

func New(p Prop) tea.Model {
	fp := filepicker.New()
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.AutoHeight = false
	fp.SetHeight(pickerHeight)
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}
	return model{picker: fp, title: p.Title, onSelect: p.OnSelect}
}

func (m model) Init() tea.Cmd {
	return m.picker.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case keys.CtrlC:
			return m, tea.Quit
		case keys.Esc:
			return m, nav.Back()
		}
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if ok, path := m.picker.DidSelectFile(msg); ok {
		if m.onSelect != nil {
			return m, m.onSelect(path)
		}
		return m, nil
	}
	if ok, _ := m.picker.DidSelectDisabledFile(msg); ok {
		m.errMsg = "File type not allowed."
	}
	return m, cmd
}

func (m model) View() tea.View {
	body := m.picker.View()
	if m.errMsg != "" {
		body += "\n" + m.errMsg
	}
	return tea.NewView(layout.Page(m.title, body, hint))
}
