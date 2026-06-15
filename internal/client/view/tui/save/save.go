package save

import (
	"gophkeeper/internal/client/view/tui/menu"

	tea "charm.land/bubbletea/v2"
)

type Choice int

const (
	LoginPassword Choice = iota
	DebitCard
	File
)

type SelectMsg struct {
	Choice Choice
}

type model struct {
	menu menu.Model
}

func New() model {
	return model{menu: menu.New("Select action", []string{"Login/password", "Debit card", "File"})}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var act menu.Action
	m.menu, act = m.menu.Update(msg)
	switch act {
	case menu.Selected:
		choice := Choice(m.menu.Cursor())
		return m, func() tea.Msg { return SelectMsg{Choice: choice} }
	case menu.Quit:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(m.menu.View())
}
