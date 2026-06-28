// Package notelist shows the notes list.
package notelist

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/home"
	"gophkeeper/internal/client/view/tui/views/notes/noteedit"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const colText = 40

const noun = "note"

type Repo interface {
	List(ctx context.Context, lastID string, limit int) ([]repository.NoteRow, error)
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
	return paginatedlist.New(paginatedlist.Config[clientmodel.Note]{
		Title:          home.LabelNotes,
		Noun:           noun,
		Header:         fmt.Sprintf("  %-*s%s", colText, "TEXT", "META"),
		AddScreen:      nav.NoteAdd,
		ConflictScreen: nav.Sync,
		Fetch: paginatedlist.Fetcher(
			func(cursor string, limit int) ([]repository.NoteRow, error) {
				return repo.List(context.Background(), cursor, limit)
			},
			func(row repository.NoteRow) (clientmodel.Note, error) { return decodeNote(vlt, row) },
			func(row repository.NoteRow) clientmodel.Note { return clientmodel.Note{ID: row.ID} },
		),
		ID:           func(n clientmodel.Note) string { return n.ID },
		Revealable:   func(n clientmodel.Note) bool { return n.Data != (clientmodel.NoteData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove: func(id string) error {
			return repo.Delete(context.Background(), id)
		},
		NewEdit: func(n clientmodel.Note) tea.Model {
			return noteedit.New(noteedit.Prop{Vault: vlt, Repo: repo, Note: n})
		},
	})
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
