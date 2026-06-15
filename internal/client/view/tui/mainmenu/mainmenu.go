package mainmenu

import (
	"gophkeeper/internal/client/view/tui/menu"
	"gophkeeper/internal/client/view/tui/nav"

	tea "charm.land/bubbletea/v2"
)

type Choice int

const (
	ViewData Choice = iota
	SaveData
	DeleteData
	Logout
)

type model struct {
	menu menu.Model
}

func New() model {
	return model{menu: menu.New("Gophkeeper", []string{"View data", "Save data", "Delete data", "Logout"})}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var act menu.Action
	m.menu, act = m.menu.Update(msg)
	switch act {
	case menu.Selected:
		switch Choice(m.menu.Cursor()) {
		case SaveData:
			return m, nav.Push(nav.Save)
		case Logout:
			return m, nav.Logout()
		}
	case menu.Back:
		return m, nav.Back()
	case menu.Quit:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.NewView(m.menu.View())
}
