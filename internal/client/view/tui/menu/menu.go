package menu

import (
	"strings"

	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/theme"

	tea "charm.land/bubbletea/v2"
)

type Action int

const (
	None Action = iota
	Selected
	Back
	Quit
)

const hint = "↑/↓ — select · enter — accept · esc — back · q — quit"

type Model struct {
	title   string
	buttons []string
	cursor  int
}

func New(title string, buttons []string) Model {
	return Model{title: title, buttons: buttons}
}

func (m Model) Cursor() int {
	return m.cursor
}

func (m Model) Update(msg tea.Msg) (Model, Action) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, None
	}
	switch key.String() {
	case "q", "ctrl+c":
		return m, Quit
	case "esc":
		return m, Back
	case "up":
		m.cursor--
		if m.cursor < 0 {
			m.cursor = len(m.buttons) - 1
		}
	case "down":
		m.cursor++
		if m.cursor >= len(m.buttons) {
			m.cursor = 0
		}
	case "enter":
		return m, Selected
	}
	return m, None
}

func (m Model) View() string {
	var b strings.Builder
	for i, label := range m.buttons {
		text := "[ " + label + " ]"
		if i == m.cursor {
			text = theme.Focused.Bold(true).Render("> " + text)
		} else {
			text = theme.Blurred.Render("  " + text)
		}
		b.WriteString(text)
		b.WriteRune('\n')
	}
	return layout.Page(m.title, b.String(), hint)
}
