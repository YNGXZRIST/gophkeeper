package credform

import (
	"gophkeeper/internal/client/view/tui/form"

	tea "charm.land/bubbletea/v2"
)

type SubmitMsg struct {
	Login    string
	Password string
}

type Model struct {
	form form.Model
}

func New() Model {
	return Model{form: form.New("send", []form.Field{
		{Placeholder: "Login", CharLimit: 256, Width: 256},
		{Placeholder: "Password", CharLimit: 128, Width: 128, Password: true},
	})}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var act form.Action
	var cmd tea.Cmd
	m.form, act, cmd = m.form.Update(msg)
	if act == form.Submit {
		values := m.form.Values()
		login, password := values[0], values[1]
		return m, func() tea.Msg {
			return SubmitMsg{Login: login, Password: password}
		}
	}
	return m, cmd
}

func (m Model) View() (string, *tea.Cursor) {
	return m.form.View(0)
}
