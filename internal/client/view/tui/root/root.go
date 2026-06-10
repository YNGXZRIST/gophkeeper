package root

import (
	"gophkeeper/internal/client/view/tui/login"
	"gophkeeper/internal/client/view/tui/register"
	"gophkeeper/internal/client/view/tui/welcome"

	tea "charm.land/bubbletea/v2"
)

type rootModel struct {
	current tea.Model
}

func New() rootModel {
	return rootModel{
		current: welcome.NewWelcomeModel(),
	}
}

func (m rootModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case welcome.SelectMsg:
		switch msg.Choice {
		case welcome.SignIn:
			m.current = login.InitialModel()
		case welcome.SignUp:
			m.current = register.InitialModel()
		}
		return m, m.current.Init()
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.current.View()
}
