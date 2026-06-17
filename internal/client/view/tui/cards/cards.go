package cards

import (
	"context"
	"fmt"
	"strings"

	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/nav"
	"gophkeeper/internal/client/view/tui/theme"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"

	tea "charm.land/bubbletea/v2"
)

const pageSize = 10

const cardsHint = "↑/↓ — move · enter — reveal · ←/→ — page · esc — back"

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

type model struct {
	vault    *vault.Vault
	client   cardv1.CardServiceClient
	items    []clientmodel.Card
	cursor   string
	history  []string
	next     string
	hasNext  bool
	selected int
	revealed bool
	errMsg   string
}

type Prop struct {
	Vault  *vault.Vault
	Client cardv1.CardServiceClient
}

func New(p Prop) model {
	return model{vault: p.Vault, client: p.Client}
}

func (m model) Init() tea.Cmd {
	return m.fetch("")
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
		return m, nil
	case tea.KeyPressMsg:
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
		case "right", "l":
			if m.hasNext {
				m.history = append(m.history, m.cursor)
				return m, m.fetch(m.next)
			}
		case "left", "h":
			if n := len(m.history); n > 0 {
				prev := m.history[n-1]
				m.history = m.history[:n-1]
				return m, m.fetch(prev)
			}
		}
	}
	return m, nil
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
	content := layout.Page("Cards", b.String(), cardsHint)
	return tea.NewView(content)
}
