// Package cardlist shows the card list.
package cardlist

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/cards/cardedit"
	"gophkeeper/internal/client/view/tui/views/home"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	colNumber = 13
	colExpiry = 8
	colHolder = 18
)

const noun = "card"

type Repo interface {
	List(ctx context.Context, lastID string, limit int) ([]repository.CardRow, error)
	Update(ctx context.Context, id string, data []byte) error
	Delete(ctx context.Context, id string) error
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
}

func New(p Prop) tea.Model {
	repo := p.Repo
	vlt := p.Vault
	return paginatedlist.New(paginatedlist.Config[clientmodel.Card]{
		Title:          home.LabelCards,
		Noun:           noun,
		Header:         fmt.Sprintf("  %-*s%-*s%-*s%s", colNumber, "NUMBER", colExpiry, "EXPIRY", colHolder, "HOLDER", "META"),
		AddScreen:      nav.CardAdd,
		ConflictScreen: nav.CardSync,
		Fetch: paginatedlist.Fetcher(
			func(cursor string, limit int) ([]repository.CardRow, error) {
				return repo.List(context.Background(), cursor, limit)
			},
			func(row repository.CardRow) (clientmodel.Card, error) { return decodeCard(vlt, row) },
			func(row repository.CardRow) clientmodel.Card { return clientmodel.Card{ID: row.ID} },
		),
		ID:           func(c clientmodel.Card) string { return c.ID },
		Revealable:   func(c clientmodel.Card) bool { return c.Data != (clientmodel.CardData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove: func(id string) error {
			return repo.Delete(context.Background(), id)
		},
		NewEdit: func(c clientmodel.Card) tea.Model {
			return cardedit.New(cardedit.Prop{Vault: vlt, Repo: repo, Card: c})
		},
	})
}

func renderItem(c clientmodel.Card) string {
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
