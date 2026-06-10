package login

import (
	"gophkeeper/internal/client/view/tui/credform"
	"gophkeeper/internal/client/view/tui/theme"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	form credform.Model
}

func InitialModel() model {
	return model{form: credform.New()}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Quit
		}

	case credform.SubmitMsg:

		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	body, cur := m.form.View()
	v := tea.NewView(theme.Title.Render("Login") + "\n\n" + body)
	v.Cursor = cur
	return v
}
