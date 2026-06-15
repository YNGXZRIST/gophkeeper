package card

import (
	"gophkeeper/internal/client/view/tui/form"
	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/nav"

	tea "charm.land/bubbletea/v2"
)

const titleOffset = 2

const cardHint = "↑/↓ — move · enter — next/save · esc — back"

type model struct {
	form form.Model
}

func New() model {
	return model{form: form.New("save", []form.Field{
		{Placeholder: "Card number", CharLimit: 256, Width: 256},
		{Placeholder: "Cardholder name", CharLimit: 256, Width: 256},
		{Placeholder: "Expiry MM/YY", CharLimit: 256, Width: 256},
		{Placeholder: "CVV", CharLimit: 4, Width: 256, Password: true},
		{Placeholder: "Meta", CharLimit: 256, Width: 256},
	})}
}

func (m model) Init() tea.Cmd {
	return m.form.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
		}
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
	return nav.Back()
}

func (m model) View() tea.View {
	body, c := m.form.View(titleOffset)
	v := tea.NewView(layout.Page("Debit card", body, cardHint))
	v.Cursor = c
	return v
}
