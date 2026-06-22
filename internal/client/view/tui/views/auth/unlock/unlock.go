package unlock

import (
	"errors"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/form"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

type model struct {
	vault   *vault.Vault
	session *auth.Session
	form    form.Model
	errMsg  string
}

func New(v *vault.Vault, session *auth.Session) tea.Model {
	return model{
		vault:   v,
		session: session,
		form: form.New("unlock", []form.Field{
			{Placeholder: "Master password", CharLimit: 128, Width: 128, Password: true},
		}),
	}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case keys.CtrlC:
			return m, tea.Quit
		case keys.Esc:
			return m, nav.Logout()
		}
	}

	var act form.Action
	var cmd tea.Cmd
	m.form, act, cmd = m.form.Update(msg)
	if act == form.Submit {
		if err := m.vault.Unlock(m.form.Values()[0], m.session); err != nil {
			if errors.Is(err, vault.ErrWrongPassword) {
				m.errMsg = "Wrong password."
			} else {
				m.errMsg = "Crashed."
			}
			return m, nil
		}
		return m, nav.Reset(nav.Home)
	}
	return m, cmd
}

func (m model) View() tea.View {
	body, cur := m.form.View(titleOffset)
	content := theme.Title.Render("Unlock") + "\n\n" + body
	if m.errMsg != "" {
		content += "\n" + theme.Error.Render(m.errMsg)
	}
	content += "\n\nenter — unlock · esc — log out"
	v := tea.NewView(content)
	v.Cursor = cur
	return v
}
