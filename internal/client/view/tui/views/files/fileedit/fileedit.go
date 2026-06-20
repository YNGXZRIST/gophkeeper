// Package fileedit is the screen for editing a stored file's metadata.
package fileedit

import (
	"context"
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/form"
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	filev1 "gophkeeper/internal/shared/proto/file/v1"

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

const label = "Edit file"

type savedMsg struct{ err error }

type Prop struct {
	Vault  *vault.Vault
	Client filev1.FileServiceClient
	File   clientmodel.File
}

type model struct {
	form   form.Model
	vault  *vault.Vault
	client filev1.FileServiceClient
	file   clientmodel.File
	errMsg string
}

func New(p Prop) tea.Model {
	return model{
		vault:  p.Vault,
		client: p.Client,
		file:   p.File,
		form: form.New("save", []form.Field{
			{Placeholder: "Meta", Value: p.File.Meta.Meta, CharLimit: 256, Width: 256},
		}),
	}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m model) submit() tea.Cmd {
	data := clientmodel.FileMeta{
		Name: m.file.Meta.Name,
		Size: m.file.Meta.Size,
		Meta: m.form.Values()[0],
	}
	vlt := m.vault
	client := m.client
	file := m.file
	return func() tea.Msg {
		raw, err := json.Marshal(data)
		if err != nil {
			return savedMsg{err: err}
		}
		ciphertext, err := vlt.Encrypt(raw)
		if err != nil {
			return savedMsg{err: err}
		}
		req := &filev1.UpdateMetaRequest{}
		req.SetId(file.ID)
		req.SetMeta(ciphertext)
		req.SetVersion(file.Version)
		if _, err := client.UpdateMeta(context.Background(), req); err != nil {
			return savedMsg{err: err}
		}
		return savedMsg{}
	}
}

func (m model) View() tea.View {
	body, c := m.form.View(titleOffset)
	content := layout.Page(label, body, form.Hint)
	if m.errMsg != "" {
		content += "\n" + m.errMsg
	}
	v := tea.NewView(content)
	v.Cursor = c
	return v
}
