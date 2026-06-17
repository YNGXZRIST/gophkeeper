package register

import (
	"context"
	"fmt"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/crypto"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/auth/internal/credform"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sessionStore interface {
	Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error)
}
type model struct {
	client userv1.UserServiceClient
	form   credform.Model
	store  sessionStore
	vault  *vault.Vault
	errMsg string
}

type Prop struct {
	Client userv1.UserServiceClient
	Store  sessionStore
	Vault  *vault.Vault
}

func New(p Prop) tea.Model {
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
		salt, err := crypto.GenerateBytes(16)
		if err != nil {
			m.errMsg = "Crashed."
			return m, nil
		}
		dek, err := crypto.GenerateBytes(32)
		if err != nil {
			m.errMsg = "Crashed."
			return m, nil
		}
		kek := crypto.DeriveKey(msg.Password, salt)
		enc, err := crypto.NewEncryptor(kek)
		if err != nil {
			m.errMsg = "Crashed."
			return m, nil
		}
		wrappedDek, err := enc.Encrypt(dek)
		if err != nil {
			m.errMsg = "Crashed."
			return m, nil
		}
		res, err := m.client.Register(context.Background(), userv1.RegisterRequest_builder{
			Login:      &msg.Login,
			Password:   &msg.Password,
			EncSalt:    salt,
			WrappedDek: wrappedDek,
		}.Build())
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				m.errMsg = "Server error. Try again later."
			case codes.AlreadyExists:
				m.errMsg = st.Message()
			default:
				m.errMsg = "Internal error. Try again later."
			}
			return m, nil
		}
		m.errMsg = ""
		_, err = m.store.Save(context.Background(), auth.Credentials{
			Login:        msg.Login,
			AccessToken:  res.GetAccessToken(),
			RefreshToken: res.GetRefreshToken(),
			EncSalt:      salt,
			WrappedDek:   wrappedDek,
		})
		if err != nil {
			fmt.Println("Error:", err)
			m.errMsg = "Client error. Try again later."
			return m, nil
		}
		if err := m.vault.UseDEK(dek); err != nil {
			m.errMsg = "Crashed."
			return m, nil
		}
		return m, nav.Reset(nav.Home)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	body, cur := m.form.View()
	content := theme.Title.Render("Register") + "\n\n" + body
	if m.errMsg != "" {
		content += "\n" + theme.Error.Render(m.errMsg)
	}
	v := tea.NewView(content)
	v.Cursor = cur
	return v
}
