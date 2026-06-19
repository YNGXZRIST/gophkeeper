// Package list shows the card list.
package list

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/cards/edit"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Column widths for the masked card list, in display runes.
const (
	colNumber = 13
	colExpiry = 8
	colHolder = 18
)

type Prop struct {
	Vault  *vault.Vault
	Client cardv1.CardServiceClient
}

func New(p Prop) tea.Model {
	return paginatedlist.New(paginatedlist.Config[clientmodel.Card]{
		Title:     "Cards",
		Noun:      "card",
		Header:    fmt.Sprintf("  %-*s%-*s%-*s%s", colNumber, "NUMBER", colExpiry, "EXPIRY", colHolder, "HOLDER", "META"),
		AddScreen: nav.CardAdd,
		Fetch: paginatedlist.Fetcher(p.list,
			func(pb *cardv1.Card) (clientmodel.Card, error) { return decodeCard(p.Vault, pb) },
			func(pb *cardv1.Card) clientmodel.Card { return clientmodel.Card{ID: pb.GetId()} },
		),
		ID:           func(c clientmodel.Card) string { return c.ID },
		Revealable:   func(c clientmodel.Card) bool { return c.Data != (clientmodel.CardData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove:       p.remove,
		NewEdit: func(c clientmodel.Card) tea.Model {
			return edit.New(edit.Prop{Vault: p.Vault, Client: p.Client, Card: c})
		},
	})
}

func (p Prop) list(cursor string, limit int) ([]*cardv1.Card, error) {
	req := &cardv1.ListRequest{}
	req.SetLastId(cursor)
	req.SetLimit(int32(limit))
	resp, err := p.Client.List(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.GetCards(), nil
}

func (p Prop) remove(id string) error {
	req := &cardv1.DeleteRequest{}
	req.SetId(id)
	_, err := p.Client.Delete(context.Background(), req)
	return err
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
