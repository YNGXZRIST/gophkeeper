package card

import (
	"fmt"
	"strings"

	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/nav"
	"gophkeeper/internal/client/view/tui/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const titleOffset = 2

const cardHint = "↑/↓ — move · enter — next/save · esc — back"

const submitLabel = "save"

type model struct {
	inputs     []textinput.Model
	focusIndex int
}

func New() model {
	placeholders := []string{"Card number", "Cardholder name", "Expiry MM/YY", "CVV", "Meta"}
	m := model{inputs: make([]textinput.Model, len(placeholders))}

	for i := range m.inputs {
		t := textinput.New()

		s := t.Styles()
		s.Cursor.Color = lipgloss.Color(theme.RoseWater)
		s.Focused.Prompt = theme.Focused
		s.Focused.Text = theme.Focused
		s.Blurred.Prompt = theme.Blurred
		t.SetStyles(s)

		t.Placeholder = placeholders[i]
		t.CharLimit = 256
		t.SetWidth(256)

		if i == 3 {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = ' '
			t.CharLimit = 4
		}
		if i == 0 {
			t.Focus()
		}

		m.inputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
		case "up":
			return m, m.moveFocus(-1)
		case "down":
			return m, m.moveFocus(1)
		case "enter":
			if m.focusIndex == len(m.inputs) {
				if !m.allFilled() {
					return m, nil
				}
				return m, m.submit()
			}
			return m, m.moveFocus(1)
		}
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	m.refreshStyles()
	return m, tea.Batch(cmds...)
}

func (m *model) moveFocus(delta int) tea.Cmd {
	m.focusIndex += delta
	if m.focusIndex > len(m.inputs) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs)
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()
	}
	return tea.Batch(cmds...)
}

func (m *model) refreshStyles() {
	for i := range m.inputs {
		s := m.inputs[i].Styles()
		if m.inputs[i].Value() != "" {
			s.Focused.Prompt = theme.Filled
			s.Blurred.Prompt = theme.Filled
		} else {
			s.Focused.Prompt = theme.Focused
			s.Blurred.Prompt = theme.Blurred
		}
		m.inputs[i].SetStyles(s)
	}
}

func (m model) allFilled() bool {
	for i := range m.inputs {
		if m.inputs[i].Value() == "" {
			return false
		}
	}
	return true
}

func (m model) submit() tea.Cmd {
	return nav.Back()
}

func (m model) View() tea.View {
	var b strings.Builder
	var c *tea.Cursor

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')

		if m.inputs[i].Focused() {
			c = m.inputs[i].Cursor()
			if c != nil {
				c.Y += titleOffset + i
			}
		}
	}

	label := "[ " + submitLabel + " ]"
	var button string
	switch {
	case !m.allFilled():
		button = theme.Blurred.Render(label)
	case m.focusIndex == len(m.inputs):
		button = theme.Focused.Bold(true).Render(label)
	default:
		button = theme.Focused.Render(label)
	}
	fmt.Fprintf(&b, "%s\n", button)

	v := tea.NewView(layout.Page("Debit card", b.String(), cardHint))
	v.Cursor = c
	return v
}
