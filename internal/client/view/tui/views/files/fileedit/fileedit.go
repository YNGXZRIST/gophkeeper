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

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

const label = "Edit file"

type savedMsg struct{ err error }

type Repo interface {
	UpdateMeta(ctx context.Context, id string, meta []byte) error
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
	File  clientmodel.File
}

type model struct {
	form   form.Model
	vault  *vault.Vault
	repo   Repo
	file   clientmodel.File
	errMsg string
}

func New(p Prop) tea.Model {
	return model{
		vault: p.Vault,
		repo:  p.Repo,
		file:  p.File,
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
		case keys.CtrlC:
			return m, tea.Quit
		case keys.Esc:
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
	repo := m.repo
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
		if err := repo.UpdateMeta(context.Background(), file.ID, ciphertext); err != nil {
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
