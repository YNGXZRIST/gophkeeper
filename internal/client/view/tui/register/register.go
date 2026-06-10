package register

import (
	"context"
	"fmt"
	"gophkeeper/internal/client/view/tui/credform"
	"gophkeeper/internal/client/view/tui/theme"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	"log"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	client userv1.UserServiceClient
	form   credform.Model
}

func InitialModel(client userv1.UserServiceClient) model {
	return model{client: client, form: credform.New()}
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
			log.Fatal(err)
		}
		fmt.Println(res)
		fmt.Println(msg.Login, msg.Password)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	body, cur := m.form.View()
	v := tea.NewView(theme.Title.Render("Register") + "\n\n" + body)
	v.Cursor = cur
	return v
}
