package nav

import tea "charm.land/bubbletea/v2"

type ScreenID int

const (
	Welcome ScreenID = iota
	Login
	Register
	MainMenu
	Save
	Card
	Unlock
)

type PushMsg struct {
	ID ScreenID
}

type ResetMsg struct {
	ID ScreenID
}

type BackMsg struct{}

type LogoutMsg struct{}

func Push(id ScreenID) tea.Cmd {
	return func() tea.Msg { return PushMsg{ID: id} }
}

func Reset(id ScreenID) tea.Cmd {
	return func() tea.Msg { return ResetMsg{ID: id} }
}

func Back() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}

func Logout() tea.Cmd {
	return func() tea.Msg { return LogoutMsg{} }
}
