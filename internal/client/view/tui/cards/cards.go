package cards

import (
	"context"
	"fmt"
	"strings"

	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/layout"
	"gophkeeper/internal/client/view/tui/nav"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"

	tea "charm.land/bubbletea/v2"
)

const pageSize = 10

const cardsHint = "←/→ — paginate · esc — back"

// loadedMsg carries the result of a single page fetch.
type loadedMsg struct {
	cursor     string
	items      []clientmodel.Card
	nextCursor string
	hasNext    bool
	err        error
}

type model struct {
	vault   *vault.Vault
	client  cardv1.CardServiceClient
	items   []clientmodel.Card
	cursor  string
	history []string
	next    string
	hasNext bool
	errMsg  string
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
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nav.Back()
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

// renderItem returns the single-line, masked representation of a card.
func renderItem(c clientmodel.Card) string {
	if c.Data == (clientmodel.CardData{}) {
		return "  [decryption failed]"
	}
	holder := c.Data.Holder
	if holder == "" {
		holder = "—"
	}
	return fmt.Sprintf("  %s  %s  %s", maskNumber(c.Data.Number), c.Data.Expiry, holder)
}

func (m model) View() tea.View {
	var b strings.Builder
	if len(m.items) == 0 {
		b.WriteString("No cards.")
	} else {
		for _, c := range m.items {
			b.WriteString(renderItem(c))
			b.WriteByte('\n')
		}
	}
	if m.errMsg != "" {
		b.WriteString("\n" + m.errMsg)
	}
	content := layout.Page("Cards", b.String(), cardsHint)
	return tea.NewView(content)
}
