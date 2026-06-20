package menu

import (
	"gophkeeper/internal/client/view/tui/components/button"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/theme"
	"strings"

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
	case keys.Q, keys.CTRL_C:
		return m, Quit
	case keys.ESC:
		return m, Back
	case keys.UP:
		m.cursor--
		if m.cursor < 0 {
			m.cursor = len(m.buttons) - 1
		}
	case keys.DOWN:
		m.cursor++
		if m.cursor >= len(m.buttons) {
			m.cursor = 0
		}
	case keys.ENTER:
		return m, Selected
	}
	return m, None
}

func (m Model) View() string {
	var b strings.Builder
	for i, label := range m.buttons {
		text := button.GetLabel(label)
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
