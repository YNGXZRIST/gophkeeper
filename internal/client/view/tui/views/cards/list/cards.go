// Package list shows the card list and its in-screen actions.
package list

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/layout"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/cards/edit"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

const pageSize = 10

const cardsHint = "↑/↓ — move · enter — reveal · a — add · e — edit · d — delete · ←/→ — page · esc — back"

const confirmHint = "delete selected card? y — yes · n — no"

// Column widths for the masked card list, in display runes.
const (
	colNumber = 13
	colExpiry = 8
	colHolder = 18
)

// loadedMsg carries the result of a single page fetch.
type loadedMsg struct {
	cursor     string
	items      []clientmodel.Card
	nextCursor string
	hasNext    bool
	err        error
}

type deletedMsg struct{ err error }

type model struct {
	vault      *vault.Vault
	client     cardv1.CardServiceClient
	items      []clientmodel.Card
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

type Prop struct {
	Vault  *vault.Vault
	Client cardv1.CardServiceClient
}

func New(p Prop) tea.Model {
	return model{vault: p.Vault, client: p.Client, loading: true, spinner: spinner.New(spinner.WithSpinner(spinner.Dot))}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.fetch(""), m.spinner.Tick)
}

// fetch loads one page starting after the given keyset cursor.
func (m model) fetch(c string) tea.Cmd {
	v, client := m.vault, m.client
	return func() tea.Msg {
		req := &cardv1.ListRequest{}
		req.SetLastId(c)
		req.SetLimit(pageSize)
		resp, err := client.List(context.Background(), req)
		if err != nil {
			return loadedMsg{cursor: c, err: err}
		}
		pbCards := resp.GetCards()
		items := make([]clientmodel.Card, 0, len(pbCards))
		for _, pb := range pbCards {
			card, decErr := decodeCard(v, pb)
			if decErr != nil {
				// keep the id so pagination still advances past a corrupt card
				card = clientmodel.Card{ID: pb.GetId()}
			}
			items = append(items, card)
		}
		msg := loadedMsg{cursor: c, items: items, hasNext: len(pbCards) == pageSize}
		if n := len(pbCards); n > 0 {
			msg.nextCursor = pbCards[n-1].GetId()
		}
		return msg
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedMsg:
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
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
		case "up":
			if m.selected > 0 {
				m.selected--
				m.revealed = false
			}
		case "down":
			if m.selected < len(m.items)-1 {
				m.selected++
				m.revealed = false
			}
		case "enter":
			if m.canReveal() {
				m.revealed = !m.revealed
			}
		case "a":
			return m, nav.Push(nav.CardAdd)
		case "e":
			if m.canDelete() {
				return m, nav.PushModel(edit.New(edit.Prop{Vault: m.vault, Client: m.client, Card: m.items[m.selected]}))
			}
		case "d":
			if m.canDelete() {
				m.confirming = true
			}
		case "right", "l":
			if m.hasNext {
				m.history = append(m.history, m.cursor)
				m.loading = true
				return m, tea.Batch(m.fetch(m.next), m.spinner.Tick)
			}
		case "left", "h":
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

// updateConfirm handles keys while a delete confirmation is pending.
func (m model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "y":
		m.confirming = false
		return m, m.remove(m.items[m.selected].ID)
	default:
		m.confirming = false
		return m, nil
	}
}

// remove deletes the card with the given id on the server.
func (m model) remove(id string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		req := &cardv1.DeleteRequest{}
		req.SetId(id)
		if _, err := client.Delete(context.Background(), req); err != nil {
			return deletedMsg{err: err}
		}
		return deletedMsg{}
	}
}

// canDelete reports whether a card is selected and can be deleted.
func (m model) canDelete() bool {
	return m.selected >= 0 && m.selected < len(m.items)
}

// canReveal reports whether the selected card carries decryptable data.
func (m model) canReveal() bool {
	if m.selected < 0 || m.selected >= len(m.items) {
		return false
	}
	return m.items[m.selected].Data != (clientmodel.CardData{})
}

// renderItem returns the single-line, masked representation of a card.
func renderItem(c clientmodel.Card) string {
	if c.Data == (clientmodel.CardData{}) {
		return "[decryption failed]"
	}
	holder := c.Data.Holder
	if holder == "" {
		holder = "—"
	}
	meta := c.Data.Meta
	if meta == "" {
		meta = "—"
	}
	return fmt.Sprintf("%-*s%-*s%-*s%s", colNumber, maskNumber(c.Data.Number), colExpiry, c.Data.Expiry, colHolder, holder, meta)
}

// renderDetail returns the full, unmasked payload of a revealed card, one field per line.
func renderDetail(c clientmodel.Card) string {
	d := c.Data
	rows := []struct{ label, value string }{
		{"Number", d.Number},
		{"Holder", d.Holder},
		{"Expiry", d.Expiry},
		{"CVV", d.CVV},
		{"Meta", d.Meta},
	}
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		value := r.value
		if value == "" {
			value = "—"
		}
		fmt.Fprintf(&b, "%s: %s", theme.Blurred.Render(r.label), theme.Filled.Render(value))
	}
	return b.String()
}

func (m model) View() tea.View {
	if m.loading {
		return tea.NewView(layout.Page("Cards", m.spinner.View()+" Loading…", cardsHint))
	}
	var b strings.Builder
	if len(m.items) == 0 {
		b.WriteString("No cards.")
	} else {
		header := fmt.Sprintf("  %-*s%-*s%-*s%s", colNumber, "NUMBER", colExpiry, "EXPIRY", colHolder, "HOLDER", "META")
		b.WriteString(theme.Blurred.Render(header) + "\n")
		for i, c := range m.items {
			marker := "  "
			if i == m.selected {
				marker = "> "
			}
			b.WriteString(marker + renderItem(c) + "\n")
		}
		if m.revealed && m.canReveal() {
			b.WriteString("\n" + renderDetail(m.items[m.selected]) + "\n")
		}
	}
	if m.errMsg != "" {
		b.WriteString("\n" + m.errMsg)
	}
	hint := cardsHint
	if m.confirming {
		hint = confirmHint
	}
	content := layout.Page("Cards", b.String(), hint)
	return tea.NewView(content)
}
