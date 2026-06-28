// Package picker is a reusable menu-driven selection screen.
package picker

import (
	"gophkeeper/internal/client/view/tui/components/menu"
	"gophkeeper/internal/client/view/tui/components/nav"

	tea "charm.land/bubbletea/v2"
)

type Item struct {
	Label  string
	Action tea.Cmd
}

type model struct {
	menu  menu.Model
	items []Item
}

func New(title string, items []Item) tea.Model {
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
	}
	return model{menu: menu.New(title, labels), items: items}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var act menu.Action
	m.menu, act = m.menu.Update(msg)
	switch act {
	case menu.Selected:
		if action := m.items[m.menu.Cursor()].Action; action != nil {
			return m, action
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
