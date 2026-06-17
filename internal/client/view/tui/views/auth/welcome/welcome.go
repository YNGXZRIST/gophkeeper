package welcome

import (
	"gophkeeper/internal/client/view/tui/components/menu"
	"gophkeeper/internal/client/view/tui/components/nav"

	tea "charm.land/bubbletea/v2"
)

type Choice int

const (
	SignIn Choice = iota
	SignUp
)

type model struct {
	menu menu.Model
}

func New() tea.Model {
	return model{menu: menu.New("Welcome to Gophkeeper!", []string{"Sign In", "Sign Up"})}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var act menu.Action
	m.menu, act = m.menu.Update(msg)
	switch act {
	case menu.Selected:
		switch Choice(m.menu.Cursor()) {
		case SignIn:
			return m, nav.Push(nav.Login)
		case SignUp:
			return m, nav.Push(nav.Register)
		}
	case menu.Back:
		return m, nav.Back()
	case menu.Quit:
		return m, tea.Quit
	default:
		return m, nil
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(m.menu.View())
}
