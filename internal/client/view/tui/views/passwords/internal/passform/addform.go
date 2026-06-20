// Package passform is the shared password form screen for adding and editing passwords.
package passform

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/form"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

type savedMsg struct{ err error }

// SaveFunc persists the encrypted password payload (Add or Update).
type SaveFunc func(ciphertext []byte) error

type Model struct {
	form   form.Model
	vault  *vault.Vault
	title  string
	save   SaveFunc
	errMsg string
}

func New(vlt *vault.Vault, title string, d clientmodel.PasswordData, save SaveFunc) Model {
	return Model{
		vault: vlt,
		title: title,
		save:  save,
		form: form.New("save", []form.Field{
			{Placeholder: "Login", Value: d.Login, CharLimit: 256, Width: 256},
			{Placeholder: "Password", Value: d.Password, CharLimit: 256, Width: 256},
			{Placeholder: "Meta", Value: d.Meta, CharLimit: 256, Width: 256},
		}),
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case keys.CTRL_C:
			return m, tea.Quit
		case keys.ESC:
			return m, nav.Back()
		}
	case savedMsg:
		if msg.err != nil {
			m.errMsg = "Save failed. " + msg.err.Error()
			return m, nil
		}
		return m, tea.Sequence(nav.Back(), nav.Reload())
	}

	var act form.Action
	var cmd tea.Cmd
	m.form, act, cmd = m.form.Update(msg)
	if act == form.Submit {
		return m, m.submit()
	}
	return m, cmd
}

func (m Model) submit() tea.Cmd {
	data := m.GetPasswordData()
	vlt := m.vault
	save := m.save
	return func() tea.Msg {
		raw, err := json.Marshal(data)
		if err != nil {
			return savedMsg{err: err}
		}
		ciphertext, err := vlt.Encrypt(raw)
		if err != nil {
			return savedMsg{err: err}
		}
		if err := save(ciphertext); err != nil {
			return savedMsg{err: err}
		}
		return savedMsg{}
	}
}

func (m Model) View() tea.View {
	body, c := m.form.View(titleOffset)
	content := layout.Page(m.title, body, form.Hint)
	if m.errMsg != "" {
		content += "\n" + m.errMsg
	}
	v := tea.NewView(content)
	v.Cursor = c
	return v
}

func (m Model) GetLogin() string    { return m.form.Values()[0] }
func (m Model) GetPassword() string { return m.form.Values()[1] }
func (m Model) GetMeta() string     { return m.form.Values()[2] }

func (m Model) GetPasswordData() clientmodel.PasswordData {
	return clientmodel.PasswordData{
		Login:    m.GetLogin(),
		Password: m.GetPassword(),
		Meta:     m.GetMeta(),
	}
}
