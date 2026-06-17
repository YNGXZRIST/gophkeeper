package form

import (
	"fmt"
	"gophkeeper/internal/client/view/tui/components/button"
	"gophkeeper/internal/client/view/tui/components/theme"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Action int

const (
	None Action = iota
	Submit
)

type Field struct {
	Placeholder string
	Value       string
	CharLimit   int
	Width       int
	Password    bool
}

type Model struct {
	inputs      []textinput.Model
	focusIndex  int
	submitLabel string
}

func New(submitLabel string, fields []Field) Model {
	m := Model{
		inputs:      make([]textinput.Model, len(fields)),
		submitLabel: submitLabel,
	}

	for i, f := range fields {
		t := textinput.New()

		s := t.Styles()
		s.Cursor.Color = lipgloss.Color(theme.RoseWater)
		s.Focused.Prompt = theme.Focused
		s.Focused.Text = theme.Focused
		s.Blurred.Prompt = theme.Blurred
		t.SetStyles(s)

		t.Placeholder = f.Placeholder
		t.CharLimit = f.CharLimit
		t.SetWidth(f.Width)
		t.SetValue(f.Value)
		if f.Password {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = ' '
		}
		if i == 0 {
			t.Focus()
		}

		m.inputs[i] = t
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, Action, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "up":
			cmd := m.moveFocus(-1)
			return m, None, cmd
		case "down":
			cmd := m.moveFocus(1)
			return m, None, cmd
		case "enter":
			if m.focusIndex == len(m.inputs) {
				if !m.allFilled() {
					return m, None, nil
				}
				return m, Submit, nil
			}
			cmd := m.moveFocus(1)
			return m, None, cmd
		}
	}

	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	m.refreshStyles()
	return m, None, tea.Batch(cmds...)
}

func (m Model) Values() []string {
	values := make([]string, len(m.inputs))
	for i := range m.inputs {
		values[i] = m.inputs[i].Value()
	}
	return values
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

func (m Model) View(cursorOffset int) (string, *tea.Cursor) {
	var b strings.Builder
	var c *tea.Cursor

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')

		if m.inputs[i].Focused() {
			c = m.inputs[i].Cursor()
			if c != nil {
				c.Y += cursorOffset + i
			}
		}
	}

	label := button.GetLabel(m.submitLabel)
	var btn string
	switch {
	case !m.allFilled():
		btn = theme.Blurred.Render(label)
	case m.focusIndex == len(m.inputs):
		btn = theme.Focused.Bold(true).Render(label)
	default:
		btn = theme.Focused.Render(label)
	}
	fmt.Fprintf(&b, "%s\n", btn)

	return b.String(), c
}
