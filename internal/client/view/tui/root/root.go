package root

import (
	"context"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/view/tui/iface"
	"gophkeeper/internal/client/view/tui/login"
	"gophkeeper/internal/client/view/tui/mainmenu"
	"gophkeeper/internal/client/view/tui/register"
	"gophkeeper/internal/client/view/tui/welcome"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
)

type rootModel struct {
	current tea.Model
	Deps
}

type Deps struct {
	Client        userv1.UserServiceClient
	SessionsStore iface.SessionStore
}

func New(deps Deps) rootModel {
	return rootModel{
		current: resolveStart(deps),
		Deps:    deps,
	}
}

func resolveStart(deps Deps) tea.Model {
	session, err := deps.SessionsStore.Get(context.Background())
	if err != nil || session == nil {
		return welcome.NewWelcomeModel()
	}
	if !session.Access.Expired(0) {
		return mainmenu.New()
	}
	if session.Refresh.Raw == "" {
		return welcome.NewWelcomeModel()
	}
	in := &userv1.RefreshRequest{}
	in.SetRefreshToken(session.Refresh.Raw)
	resp, err := deps.Client.Refresh(context.Background(), in)
	if err != nil {
		return welcome.NewWelcomeModel()
	}
	if _, err := deps.SessionsStore.Save(context.Background(), auth.Credentials{
		Login:        session.Login,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		EncSalt:      session.EncSalt,
		WrappedDek:   session.WrappedDek,
	}); err != nil {
		return welcome.NewWelcomeModel()
	}
	return mainmenu.New()
}

func (m rootModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case welcome.SelectMsg:
		switch msg.Choice {
		case welcome.SignIn:
			m.current = login.InitialModel(m.Client, m.SessionsStore)
		case welcome.SignUp:
			m.current = register.InitialModel(m.Client, m.SessionsStore)
		}
		return m, m.current.Init()
	case login.SuccessMsg, register.SuccessMsg:
		m.current = mainmenu.New()
		return m, m.current.Init()
	case login.BackMsg, register.BackMsg:
		m.current = welcome.NewWelcomeModel()
		return m, m.current.Init()
	case mainmenu.SelectMsg:
		switch msg.Choice {
		case mainmenu.Logout:
			if err := m.SessionsStore.Clear(context.Background()); err != nil {
				return m, nil
			}
			m.current = welcome.NewWelcomeModel()
			return m, m.current.Init()
		}
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.current.View()
}
