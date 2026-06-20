// Package paginatedlist is the generic, keyset-paginated list screen shared by
// every secret type. A concrete type wires itself in through Config; this
// component owns navigation, reveal, delete confirmation, paging and spinner.
package paginatedlist

import (
	"gophkeeper/internal/client/view/tui/components/keys"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

const pageSize = 10

const listHint = "↑/↓ — move · enter — reveal · a — add · e — edit · d — delete · ←/→ — page · esc — back"

// Config wires a concrete secret type T into the generic list screen. Every
// callback is optional; a nil one disables the action it would drive.
type Config[T any] struct {
	// Title is the screen heading and, lowercased, the empty-state message.
	Title string
	// Noun is the singular word used in the delete confirmation prompt.
	Noun string
	// Header is the column header line, including the two-space marker gutter.
	Header string
	// AddScreen is the screen pushed when the user presses "a".
	AddScreen nav.ScreenID
	// Fetch loads one keyset page starting after cursor, returning up to limit items.
	Fetch func(cursor string, limit int) ([]T, error)
	// ID returns the server id of an item, used for paging and deletion.
	ID func(T) string
	// Revealable reports whether an item decrypted successfully and can be shown.
	Revealable func(T) bool
	// RenderItem returns the masked single-line representation of a valid item.
	RenderItem func(T) string
	// RenderDetail returns the full unmasked payload of a revealed item.
	RenderDetail func(T) string
	// Remove deletes the item with the given id on the server.
	Remove func(id string) error
	// NewEdit builds the edit screen for the given item.
	NewEdit func(T) tea.Model
	// NewDownload builds the download screen for the given item; when set, the
	// list offers a "save" action.
	NewDownload func(T) tea.Model
}

type loadedMsg[T any] struct {
	cursor     string
	items      []T
	nextCursor string
	hasNext    bool
	err        error
}

type deletedMsg struct{ err error }

type model[T any] struct {
	cfg        Config[T]
	items      []T
	cursor     string
	history    []string
	next       string
	hasNext    bool
	selected   int
	revealed   bool
	confirming bool
	loading    bool
	spinner    spinner.Model
	errMsg     string
}

// New builds the list screen for the secret type described by cfg. Every cfg
// callback is optional: a nil one is skipped at the call site, never invoked.
func New[T any](cfg Config[T]) tea.Model {
	return model[T]{cfg: cfg, loading: true, spinner: spinner.New(spinner.WithSpinner(spinner.Dot))}
}

// Fetcher builds a Config.Fetch from a raw page loader and a decoder. Items
// that fail to decode are replaced by fallback so paging advances past them.
func Fetcher[PB, T any](
	list func(cursor string, limit int) ([]PB, error),
	decode func(PB) (T, error),
	fallback func(PB) T,
) func(cursor string, limit int) ([]T, error) {
	return func(cursor string, limit int) ([]T, error) {
		pbs, err := list(cursor, limit)
		if err != nil {
			return nil, err
		}
		items := make([]T, 0, len(pbs))
		for _, pb := range pbs {
			v, decErr := decode(pb)
			if decErr != nil {
				v = fallback(pb)
			}
			items = append(items, v)
		}
		return items, nil
	}
}

func (m model[T]) Init() tea.Cmd {
	return tea.Batch(m.fetch(""), m.spinner.Tick)
}

func (m model[T]) fetch(c string) tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		if cfg.Fetch == nil {
			return loadedMsg[T]{cursor: c}
		}
		items, err := cfg.Fetch(c, pageSize)
		if err != nil {
			return loadedMsg[T]{cursor: c, err: err}
		}
		msg := loadedMsg[T]{cursor: c, items: items, hasNext: len(items) == pageSize}
		if n := len(items); n > 0 && cfg.ID != nil {
			msg.nextCursor = cfg.ID(items[n-1])
		}
		return msg
	}
}

func (m model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg[T]:
		m.loading = false
		if msg.err != nil {
			m.errMsg = "Could not load."
			return m, nil
		}
		m.errMsg = ""
		m.cursor = msg.cursor
		m.items = msg.items
		m.next = msg.nextCursor
		m.hasNext = msg.hasNext
		m.selected = 0
		m.revealed = false
		m.confirming = false
		return m, nil
	case deletedMsg:
		if msg.err != nil {
			m.errMsg = "Delete failed."
			return m, nil
		}
		return m, m.fetch(m.cursor)
	case nav.ReloadMsg:
		m.loading = true
		return m, tea.Batch(m.fetch(m.cursor), m.spinner.Tick)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			return m, cmd
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.confirming {
			return m.updateConfirm(msg)
		}
		switch msg.String() {
		case keys.ESC:
			return m, nav.Back()
		case keys.UP:
			if m.selected > 0 {
				m.selected--
				m.revealed = false
			}
		case keys.DOWN:
			if m.selected < len(m.items)-1 {
				m.selected++
				m.revealed = false
			}
		case keys.ENTER:
			if m.canReveal() {
				m.revealed = !m.revealed
			}
		case keys.A:
			return m, nav.Push(m.cfg.AddScreen)
		case keys.E:
			if m.canDelete() && m.cfg.NewEdit != nil {
				return m, nav.PushModel(m.cfg.NewEdit(m.items[m.selected]))
			}
		case keys.S:
			if m.canDelete() && m.cfg.NewDownload != nil {
				return m, nav.PushModel(m.cfg.NewDownload(m.items[m.selected]))
			}
		case keys.D:
			if m.canDelete() && m.cfg.Remove != nil {
				m.confirming = true
			}
		case keys.RIGHT, keys.L:
			if m.hasNext {
				m.history = append(m.history, m.cursor)
				m.loading = true
				return m, tea.Batch(m.fetch(m.next), m.spinner.Tick)
			}
		case keys.LEFT, keys.H:
			if n := len(m.history); n > 0 {
				prev := m.history[n-1]
				m.history = m.history[:n-1]
				m.loading = true
				return m, tea.Batch(m.fetch(prev), m.spinner.Tick)
			}
		}
	}
	return m, nil
}

func (m model[T]) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keys.Y:
		m.confirming = false
		if m.cfg.ID == nil {
			return m, nil
		}
		return m, m.remove(m.cfg.ID(m.items[m.selected]))
	default:
		m.confirming = false
		return m, nil
	}
}

func (m model[T]) remove(id string) tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		if cfg.Remove == nil {
			return deletedMsg{}
		}
		if err := cfg.Remove(id); err != nil {
			return deletedMsg{err: err}
		}
		return deletedMsg{}
	}
}

func (m model[T]) canDelete() bool {
	return m.selected >= 0 && m.selected < len(m.items)
}

func (m model[T]) revealable(item T) bool {
	return m.cfg.Revealable == nil || m.cfg.Revealable(item)
}

func (m model[T]) canReveal() bool {
	if m.selected < 0 || m.selected >= len(m.items) {
		return false
	}
	return m.revealable(m.items[m.selected])
}

func (m model[T]) renderItem(item T) string {
	if !m.revealable(item) {
		return "[decryption failed]"
	}
	if m.cfg.RenderItem == nil {
		return ""
	}
	return m.cfg.RenderItem(item)
}

func (m model[T]) View() tea.View {
	if m.loading {
		return tea.NewView(layout.Page(m.cfg.Title, m.spinner.View()+" Loading…", listHint))
	}
	var b strings.Builder
	if len(m.items) == 0 {
		b.WriteString("No " + strings.ToLower(m.cfg.Title) + ".")
	} else {
		b.WriteString(theme.Blurred.Render(m.cfg.Header) + "\n")
		for i, item := range m.items {
			marker := "  "
			if i == m.selected {
				marker = "> "
			}
			b.WriteString(marker + m.renderItem(item) + "\n")
		}
		if m.revealed && m.canReveal() && m.cfg.RenderDetail != nil {
			b.WriteString("\n" + m.cfg.RenderDetail(m.items[m.selected]) + "\n")
		}
	}
	if m.errMsg != "" {
		b.WriteString("\n" + m.errMsg)
	}
	hint := listHint
	if m.cfg.NewDownload != nil {
		hint += " · s — download"
	}
	if m.confirming {
		hint = "delete selected " + m.cfg.Noun + "? y — yes · n — no"
	}
	return tea.NewView(layout.Page(m.cfg.Title, b.String(), hint))
}
