package credform

import (
	"fmt"
	"gophkeeper/internal/client/view/tui/theme"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type SubmitMsg struct {
	Login    string
	Password string
}

const submitLabel = "send"

type Model struct {
	inputs     []textinput.Model
	focusIndex int // 0,1 = inputs; len(inputs) = submit button
}

func New() Model {
	m := Model{
		inputs: make([]textinput.Model, 2),
	}

	for i := range m.inputs {
		t := textinput.New()

		s := t.Styles()
		s.Cursor.Color = lipgloss.Color(theme.RoseWater)
		s.Focused.Prompt = theme.Focused
		s.Focused.Text = theme.Focused
		s.Blurred.Prompt = theme.Blurred
		t.SetStyles(s)

		switch i {
		case 0:
			t.Placeholder = "Login"
			t.CharLimit = 256
			t.SetWidth(256)
			t.Focus()
		case 1:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = ' '
			t.CharLimit = 128
			t.SetWidth(128)

		}

		m.inputs[i] = t
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
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

func (m *Model) moveFocus(delta int) tea.Cmd {
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

func (m *Model) refreshStyles() {
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

func (m Model) allFilled() bool {
	for i := range m.inputs {
		if m.inputs[i].Value() == "" {
			return false
		}
	}
	return true
}

func (m Model) submit() tea.Cmd {
	login, password := m.inputs[0].Value(), m.inputs[1].Value()
	return func() tea.Msg {
		return SubmitMsg{Login: login, Password: password}
	}
}

func (m Model) View() (string, *tea.Cursor) {
	var b strings.Builder
	var c *tea.Cursor

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')

		if m.inputs[i].Focused() {
			c = m.inputs[i].Cursor()
			if c != nil {
				c.Y += i
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
	fmt.Fprintf(&b, "\n%s\n", button)

	return b.String(), c
}
