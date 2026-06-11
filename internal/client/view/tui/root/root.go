package root

import (
	"gophkeeper/internal/client/view/tui/iface"
	"gophkeeper/internal/client/view/tui/login"
	"gophkeeper/internal/client/view/tui/register"
	"gophkeeper/internal/client/view/tui/welcome"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
)

type rootModel struct {
	client  userv1.UserServiceClient
	current tea.Model
	Deps
}

type Deps struct {
	Client        userv1.UserServiceClient
	SessionsStore iface.SessionStore
}

func New(deps Deps) rootModel {
	return rootModel{
		current: welcome.NewWelcomeModel(),
		Deps:    deps,
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
			m.current = login.InitialModel(m.client, m.SessionsStore)
		case welcome.SignUp:
			m.current = register.InitialModel(m.client, m.SessionsStore)
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
