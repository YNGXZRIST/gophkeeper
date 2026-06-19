// Package home is the main menu shown after the vault is unlocked.
package home

import (
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/picker"

	tea "charm.land/bubbletea/v2"
)

func New() tea.Model {
	return picker.New("Gophkeeper", []picker.Item{
		{Label: "Cards", Action: nav.Push(nav.Cards)},
		{Label: "Files"},
		{Label: "Passwords", Action: nav.Push(nav.Passwords)},
		{Label: "Logout", Action: nav.Logout()},
	})
}
