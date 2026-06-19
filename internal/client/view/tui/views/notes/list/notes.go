// Package list shows the notes list.
package list

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/notes/edit"
	notev1 "gophkeeper/internal/shared/proto/note/v1"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Column width for the note list, in display runes.
const colText = 40

type Prop struct {
	Vault  *vault.Vault
	Client notev1.NoteServiceClient
}

func New(p Prop) tea.Model {
	return paginatedlist.New(paginatedlist.Config[clientmodel.Note]{
		Title:     "Notes",
		Noun:      "note",
		Header:    fmt.Sprintf("  %-*s%s", colText, "TEXT", "META"),
		AddScreen: nav.NoteAdd,
		Fetch: paginatedlist.Fetcher(p.list,
			func(pb *notev1.Note) (clientmodel.Note, error) { return decodeNote(p.Vault, pb) },
			func(pb *notev1.Note) clientmodel.Note { return clientmodel.Note{ID: pb.GetId()} },
		),
		ID:           func(n clientmodel.Note) string { return n.ID },
		Revealable:   func(n clientmodel.Note) bool { return n.Data != (clientmodel.NoteData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove:       p.remove,
		NewEdit: func(n clientmodel.Note) tea.Model {
			return edit.New(edit.Prop{Vault: p.Vault, Client: p.Client, Note: n})
		},
	})
}

func (p Prop) list(cursor string, limit int) ([]*notev1.Note, error) {
	req := &notev1.ListRequest{}
	req.SetLastId(cursor)
	req.SetLimit(int32(limit))
	resp, err := p.Client.List(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.GetNotes(), nil
}

func (p Prop) remove(id string) error {
	req := &notev1.DeleteRequest{}
	req.SetId(id)
	_, err := p.Client.Delete(context.Background(), req)
	return err
}

func renderItem(n clientmodel.Note) string {
	meta := n.Data.Meta
	if meta == "" {
		meta = "—"
	}
	return fmt.Sprintf("%-*s%s", colText, snippet(n.Data.Text), meta)
}

func renderDetail(n clientmodel.Note) string {
	d := n.Data
	rows := []struct{ label, value string }{
		{"Text", d.Text},
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
