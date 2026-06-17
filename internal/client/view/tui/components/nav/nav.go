package nav

import tea "charm.land/bubbletea/v2"

type ScreenID int

const (
	Welcome ScreenID = iota
	Login
	Register
	Home
	Unlock
	Cards
	CardAdd
)

type PushMsg struct {
	ID ScreenID
}

type PushModelMsg struct {
	Model tea.Model
}

type ResetMsg struct {
	ID ScreenID
}

type BackMsg struct{}

type LogoutMsg struct{}

type ReloadMsg struct{}

func Push(id ScreenID) tea.Cmd {
	return func() tea.Msg { return PushMsg{ID: id} }
}

func PushModel(m tea.Model) tea.Cmd {
	return func() tea.Msg { return PushModelMsg{Model: m} }
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

func Reload() tea.Cmd {
	return func() tea.Msg { return ReloadMsg{} }
}
