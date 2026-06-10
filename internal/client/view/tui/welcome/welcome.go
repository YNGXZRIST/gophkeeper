package welcome

import (
	"strings"

	"gophkeeper/internal/client/view/tui/theme"

	tea "charm.land/bubbletea/v2"
)

// Choice identifies which button the user picked on the welcome screen.
type Choice int

const (
	SignIn Choice = iota
	SignUp
)

// SelectMsg is emitted when the user confirms a button. The embedding screen
// reads Choice to decide which screen to route to.
type SelectMsg struct {
	Choice Choice
}

type model struct {
	buttons []string
	cursor  int
}

// NewWelcomeModel builds the welcome screen with the sign-in / sign-up buttons.
func NewWelcomeModel() model {
	return model{buttons: []string{"Sign In", "Sign Up"}, cursor: 0}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
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
	b.WriteString(theme.Title.Render("Welcome to Gophkeeper!"))
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
