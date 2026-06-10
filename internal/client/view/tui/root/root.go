package root

import (
	"gophkeeper/internal/client/view/tui/login"
	"gophkeeper/internal/client/view/tui/register"
	"gophkeeper/internal/client/view/tui/welcome"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
)

type rootModel struct {
	client  userv1.UserServiceClient
	current tea.Model
}

func New(client userv1.UserServiceClient) rootModel {
	return rootModel{
		client:  client,
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
			m.current = login.InitialModel(m.client)
		case welcome.SignUp:
			m.current = register.InitialModel(m.client)
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
