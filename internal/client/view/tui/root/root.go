package root

import (
	"context"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/card"
	"gophkeeper/internal/client/view/tui/iface"
	"gophkeeper/internal/client/view/tui/login"
	"gophkeeper/internal/client/view/tui/mainmenu"
	"gophkeeper/internal/client/view/tui/nav"
	"gophkeeper/internal/client/view/tui/register"
	"gophkeeper/internal/client/view/tui/save"
	"gophkeeper/internal/client/view/tui/unlock"
	"gophkeeper/internal/client/view/tui/welcome"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
)

type rootModel struct {
	current tea.Model
	stack   []tea.Model
	Deps
}

type Deps struct {
	Client        userv1.UserServiceClient
	SessionsStore iface.SessionStore
	Vault         *vault.Vault
}

func New(deps Deps) rootModel {
	return rootModel{
		current: build(deps, resolveStart(deps)),
		Deps:    deps,
	}
}

func build(deps Deps, id nav.ScreenID) tea.Model {
	switch id {
	case nav.Login:
		return login.InitialModel(login.Prop{Client: deps.Client, Store: deps.SessionsStore, Vault: deps.Vault})
	case nav.Register:
		return register.InitialModel(register.Prop{Client: deps.Client, Store: deps.SessionsStore, Vault: deps.Vault})
	case nav.Unlock:
		session, err := deps.SessionsStore.Get(context.Background())
		if err != nil {
			return welcome.NewWelcomeModel()
		}
		return unlock.InitialModel(deps.Vault, session)
	case nav.MainMenu:
		return mainmenu.New()
	case nav.Save:
		return save.New()
	case nav.Card:
		return card.New()
	default:
		return welcome.NewWelcomeModel()
	}
}

func resolveStart(deps Deps) nav.ScreenID {
	session, err := deps.SessionsStore.Get(context.Background())
	if err != nil || session == nil {
		return nav.Welcome
	}
	if !session.Access.Expired(0) {
		if deps.Vault.Locked() {
			return nav.Unlock
		}
		return nav.MainMenu
	}
	if session.Refresh.Raw == "" {
		return nav.Welcome
	}
	in := &userv1.RefreshRequest{}
	in.SetRefreshToken(session.Refresh.Raw)
	resp, err := deps.Client.Refresh(context.Background(), in)
	if err != nil {
		return nav.Welcome
	}
	if _, err := deps.SessionsStore.Save(context.Background(), auth.Credentials{
		Login:        session.Login,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		EncSalt:      session.EncSalt,
		WrappedDek:   session.WrappedDek,
	}); err != nil {
		return nav.Welcome
	}
	if deps.Vault.Locked() {
		return nav.Unlock
	}
	return nav.MainMenu
}

func (m rootModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case nav.PushMsg:
		m.stack = append(m.stack, m.current)
		m.current = build(m.Deps, msg.ID)
		return m, m.current.Init()
	case nav.ResetMsg:
		m.stack = nil
		m.current = build(m.Deps, msg.ID)
		return m, m.current.Init()
	case nav.BackMsg:
		if n := len(m.stack); n > 0 {
			m.current = m.stack[n-1]
			m.stack = m.stack[:n-1]
		}
		return m, nil
	case nav.LogoutMsg:
		if err := m.SessionsStore.Clear(context.Background()); err != nil {
			return m, nil
		}
		m.Vault.Lock()
		m.stack = nil
		m.current = build(m.Deps, nav.Welcome)
		return m, m.current.Init()
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.current.View()
}
