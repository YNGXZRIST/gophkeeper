package mainmenu

import (
	"gophkeeper/internal/client/view/tui/theme"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type Choice int

const (
	ViewData Choice = iota
	SaveData
	DeleteData
	Logout
)

type SelectMsg struct {
	Choice Choice
}

type model struct {
	buttons []string
	cursor  int
}

func New() model {
	return model{
		buttons: []string{"View data", "Save data", "Delete data", "Logout"},
		cursor:  0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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
			return m, m.selectCmd()
		}
	}
	return m, nil
}

func (m model) selectCmd() tea.Cmd {
	choice := Choice(m.cursor)
	return func() tea.Msg {
		return SelectMsg{Choice: choice}
	}
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(theme.Title.Render("Select action"))
	b.WriteString("\n\n")

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

	b.WriteString("\n")
	b.WriteString(theme.Blurred.Render("↑/↓ — select · enter — accept · q — quit"))

	return tea.NewView(b.String())
}
