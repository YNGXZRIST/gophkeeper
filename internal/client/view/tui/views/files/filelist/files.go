// Package list shows the files list.
package filelist

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/components/paginatedlist"
	"gophkeeper/internal/client/view/tui/components/theme"
	"gophkeeper/internal/client/view/tui/views/files/filedownload"
	"gophkeeper/internal/client/view/tui/views/files/fileedit"
	"gophkeeper/internal/client/view/tui/views/home"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Column widths for the file list, in display runes.
const (
	colName = 28
	colSize = 12
)

const noun = "file"

type Prop struct {
	Vault  *vault.Vault
	Client filev1.FileServiceClient
}

func New(p Prop) tea.Model {
	return paginatedlist.New(paginatedlist.Config[clientmodel.File]{
		Title:     home.LabelFiles,
		Noun:      noun,
		Header:    fmt.Sprintf("  %-*s%-*s%s", colName, "NAME", colSize, "SIZE", "META"),
		AddScreen: nav.FileUpload,
		Fetch: paginatedlist.Fetcher(p.list,
			func(pb *filev1.File) (clientmodel.File, error) { return decodeFile(p.Vault, pb) },
			func(pb *filev1.File) clientmodel.File { return clientmodel.File{ID: pb.GetId()} },
		),
		ID:           func(f clientmodel.File) string { return f.ID },
		Revealable:   func(f clientmodel.File) bool { return f.Meta != (clientmodel.FileMeta{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove:       p.remove,
		NewEdit: func(f clientmodel.File) tea.Model {
			return fileedit.New(fileedit.Prop{Vault: p.Vault, Client: p.Client, File: f})
		},
		NewDownload: func(f clientmodel.File) tea.Model {
			return filedownload.New(filedownload.Prop{Vault: p.Vault, Client: p.Client, File: f})
		},
	})
}

func (p Prop) list(cursor string, limit int) ([]*filev1.File, error) {
	req := &filev1.ListRequest{}
	req.SetLastId(cursor)
	req.SetLimit(int32(limit))
	resp, err := p.Client.List(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.GetFiles(), nil
}

func (p Prop) remove(id string) error {
	req := &filev1.DeleteRequest{}
	req.SetId(id)
	_, err := p.Client.Delete(context.Background(), req)
	return err
}

func renderItem(f clientmodel.File) string {
	name := f.Meta.Name
	if name == "" {
		name = "—"
	}
	meta := f.Meta.Meta
	if meta == "" {
		meta = "—"
	}
	return fmt.Sprintf("%-*s%-*s%s", colName, name, colSize, humanSize(f.Meta.Size), meta)
}

func renderDetail(f clientmodel.File) string {
	rows := []struct{ label, value string }{
		{"Name", f.Meta.Name},
		{"Size", humanSize(f.Meta.Size)},
		{"Meta", f.Meta.Meta},
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
