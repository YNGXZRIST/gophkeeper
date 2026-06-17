package card

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/form"
	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/nav"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	"gophkeeper/pkg/luhn"

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

const cardHint = "↑/↓ — move · enter — next/save · esc — back"

type savedMsg struct{ err error }

type model struct {
	form   form.Model
	vault  *vault.Vault
	client cardv1.CardServiceClient
	errMsg string
}

type Prop struct {
	Vault  *vault.Vault
	Client cardv1.CardServiceClient
}

func New(p Prop) model {
	return model{
		vault:  p.Vault,
		client: p.Client,
		form: form.New("save", []form.Field{
			{Placeholder: "Card number", CharLimit: 256, Width: 256},
			{Placeholder: "Cardholder name", CharLimit: 256, Width: 256},
			{Placeholder: "Expiry MM/YY", CharLimit: 256, Width: 256},
			{Placeholder: "CVV", CharLimit: 4, Width: 256, Password: true},
			{Placeholder: "Meta", CharLimit: 256, Width: 256},
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
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
		}
	case savedMsg:
		if msg.err != nil {
			m.errMsg = "Save failed. " + msg.err.Error()
			return m, nil
		}
		return m, nav.Back()
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
	cNumber := m.form.Values()[0]
	return func() tea.Msg {
		if !luhn.Validate(cNumber) {
			return savedMsg{err: errors.New("not valid card number")}
		}
		data := clientmodel.CardData{
			Number: cNumber,
			Holder: m.form.Values()[1],
			Expiry: m.form.Values()[2],
			CVV:    m.form.Values()[3],
			Meta:   m.form.Values()[4],
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return savedMsg{err: err}
		}
		ciphertext, err := m.vault.Encrypt(raw)
		if err != nil {
			return savedMsg{err: err}
		}
		req := &cardv1.AddRequest{}
		req.SetData(ciphertext)
		if _, err := m.client.Add(context.Background(), req); err != nil {
			return savedMsg{err: err}
		}
		return savedMsg{}
	}
}

func (m model) View() tea.View {
	body, c := m.form.View(titleOffset)
	content := layout.Page("Debit card", body, cardHint)
	if m.errMsg != "" {
		content += "\n" + m.errMsg
	}
	v := tea.NewView(content)
	v.Cursor = c
	return v
}
