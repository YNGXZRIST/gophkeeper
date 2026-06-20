// Package list shows the passwords list.
package passwordlist

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/home"
	"gophkeeper/internal/client/view/tui/views/passwords/passwordedit"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Column widths for the masked password list, in display runes.
const (
	colLogin    = 24
	colPassword = 12
)

const noun = "password"

type Prop struct {
	Vault  *vault.Vault
	Client passwordv1.PasswordServiceClient
}

func New(p Prop) tea.Model {
	return paginatedlist.New(paginatedlist.Config[clientmodel.Password]{
		Title:     home.LabelPasswords,
		Noun:      noun,
		Header:    fmt.Sprintf("  %-*s%-*s%s", colLogin, "LOGIN", colPassword, "PASSWORD", "META"),
		AddScreen: nav.PasswordAdd,
		Fetch: paginatedlist.Fetcher(p.list,
			func(pb *passwordv1.Password) (clientmodel.Password, error) { return decodePassword(p.Vault, pb) },
			func(pb *passwordv1.Password) clientmodel.Password { return clientmodel.Password{ID: pb.GetId()} },
		),
		ID:           func(pw clientmodel.Password) string { return pw.ID },
		Revealable:   func(pw clientmodel.Password) bool { return pw.Data != (clientmodel.PasswordData{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove:       p.remove,
		NewEdit: func(pw clientmodel.Password) tea.Model {
			return passwordedit.New(passwordedit.Prop{Vault: p.Vault, Client: p.Client, Password: pw})
		},
	})
}

func (p Prop) list(cursor string, limit int) ([]*passwordv1.Password, error) {
	req := &passwordv1.ListRequest{}
	req.SetLastId(cursor)
	req.SetLimit(int32(limit))
	resp, err := p.Client.List(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.GetPasswords(), nil
}

func (p Prop) remove(id string) error {
	req := &passwordv1.DeleteRequest{}
	req.SetId(id)
	_, err := p.Client.Delete(context.Background(), req)
	return err
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
