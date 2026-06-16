package login

import (
	"context"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/credform"
	"gophkeeper/internal/client/view/tui/iface"
	"gophkeeper/internal/client/view/tui/nav"
	"gophkeeper/internal/client/view/tui/theme"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type model struct {
	client userv1.UserServiceClient
	form   credform.Model
	store  iface.SessionStore
	vault  *vault.Vault
	errMsg string
}

type Prop struct {
	Client userv1.UserServiceClient
	Store  iface.SessionStore
	Vault  *vault.Vault
}

func InitialModel(p Prop) model {
	return model{client: p.Client, form: credform.New(), store: p.Store, vault: p.Vault}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
		}

	case credform.SubmitMsg:
		res, err := m.client.Login(context.Background(), userv1.LoginRequest_builder{Login: &msg.Login, Password: &msg.Password}.Build())
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				m.errMsg = "Server error. Try again later."
			case codes.Unauthenticated:
				m.errMsg = "Invalid login or password."
			default:
				m.errMsg = "Internal error. Try again later."
			}
			return m, nil
		}
		m.errMsg = ""
		session, err := m.store.Save(context.Background(), auth.Credentials{
			Login:        res.GetUser().GetLogin(),
			AccessToken:  res.GetAccessToken(),
			RefreshToken: res.GetRefreshToken(),
			EncSalt:      res.GetEncSalt(),
			WrappedDek:   res.GetWrappedDek(),
		})
		if err != nil {
			m.errMsg = "Client error. Try again later."
			return m, nil
		}
		if err := m.vault.Unlock(msg.Password, session); err != nil {
			m.errMsg = "Wrong password."
			return m, nil
		}
		return m, nav.Reset(nav.MainMenu)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	body, cur := m.form.View()
	content := theme.Title.Render("Login") + "\n" + body
	if m.errMsg != "" {
		content += "\n" + theme.Error.Render(m.errMsg)
	}
	v := tea.NewView(content)
	v.Cursor = cur
	return v
}
