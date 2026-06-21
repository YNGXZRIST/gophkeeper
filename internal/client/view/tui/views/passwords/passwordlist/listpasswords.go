// Package passwordlist shows the passwords list.
package passwordlist

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
	"gophkeeper/internal/client/view/tui/views/passwords/passwordedit"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	colLogin    = 24
	colPassword = 12
)

const noun = "password"

type Repo interface {
	List(ctx context.Context, lastID string, limit int) ([]repository.PasswordRow, error)
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
	return paginatedlist.New(paginatedlist.Config[clientmodel.Password]{
		Title:          home.LabelPasswords,
		Noun:           noun,
		Header:         fmt.Sprintf("  %-*s%-*s%s", colLogin, "LOGIN", colPassword, "PASSWORD", "META"),
		AddScreen:      nav.PasswordAdd,
		ConflictScreen: nav.PasswordSync,
		Fetch: paginatedlist.Fetcher(
			func(cursor string, limit int) ([]repository.PasswordRow, error) {
				return repo.List(context.Background(), cursor, limit)
			},
			func(row repository.PasswordRow) (clientmodel.Password, error) { return decodePassword(vlt, row) },
			func(row repository.PasswordRow) clientmodel.Password { return clientmodel.Password{ID: row.ID} },
		),
		ID:           func(pw clientmodel.Password) string { return pw.ID },
		Revealable:   func(pw clientmodel.Password) bool { return pw.Data != (clientmodel.PasswordData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove: func(id string) error {
			return repo.Delete(context.Background(), id)
		},
		NewEdit: func(pw clientmodel.Password) tea.Model {
			return passwordedit.New(passwordedit.Prop{Vault: vlt, Repo: repo, Password: pw})
		},
	})
}

func renderItem(pw clientmodel.Password) string {
	login := pw.Data.Login
	if login == "" {
		login = "—"
	}
	meta := pw.Data.Meta
	if meta == "" {
		meta = "—"
	}
	return fmt.Sprintf("%-*s%-*s%s", colLogin, login, colPassword, maskPassword(pw.Data.Password), meta)
}

func renderDetail(pw clientmodel.Password) string {
	d := pw.Data
	rows := []struct{ label, value string }{
		{"Login", d.Login},
		{"Password", d.Password},
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
