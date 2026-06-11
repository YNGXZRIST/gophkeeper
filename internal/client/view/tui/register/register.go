package register

import (
	"context"
	"gophkeeper/internal/client/view/tui/credform"
	"gophkeeper/internal/client/view/tui/iface"
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
	errMsg string
}

func InitialModel(client userv1.UserServiceClient, store iface.SessionStore) model {
	return model{client: client, form: credform.New(), store: store}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		}

	case credform.SubmitMsg:
		res, err := m.client.Register(context.Background(), userv1.RegisterRequest_builder{Login: &msg.Login, Password: &msg.Password}.Build())
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
		_, err = m.store.Save(context.Background(), msg.Login, res.GetAccessToken(), res.GetRefreshToken())
		if err != nil {
			m.errMsg = "Client error. Try again later."
			return m, nil
		}
		return m, tea.Quit
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
