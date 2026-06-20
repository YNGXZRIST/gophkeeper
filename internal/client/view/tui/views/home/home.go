// Package home is the main menu shown after the vault is unlocked.
package home

import (
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/picker"

	tea "charm.land/bubbletea/v2"
)

const title = "Gophkeeper"
const LabelCards = "Cards"
const LabelFiles = "Files"
const LabelPasswords = "Passwords"
const LabelNotes = "Notes"
const LabelLogout = "Logout"

func New() tea.Model {
	return picker.New(title, []picker.Item{
		{Label: LabelCards, Action: nav.Push(nav.Cards)},
		{Label: LabelFiles, Action: nav.Push(nav.Files)},
		{Label: LabelPasswords, Action: nav.Push(nav.Passwords)},
		{Label: LabelNotes, Action: nav.Push(nav.Notes)},
		{Label: LabelLogout, Action: nav.Logout()},
	})
}
