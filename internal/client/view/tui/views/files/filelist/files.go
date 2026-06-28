// Package filelist shows the files list.
package filelist

import (
	"context"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
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

const (
	colName = 28
	colSize = 12
)

const noun = "file"

type Repo interface {
	List(ctx context.Context, lastID string, limit int) ([]repository.FileRow, error)
	UpdateMeta(ctx context.Context, id string, meta []byte) error
	Delete(ctx context.Context, id string) error
}

type Prop struct {
	Vault  *vault.Vault
	Repo   Repo
	Client filev1.FileServiceClient
}

func New(p Prop) tea.Model {
	repo := p.Repo
	vlt := p.Vault
	client := p.Client
	return paginatedlist.New(paginatedlist.Config[clientmodel.File]{
		Title:          home.LabelFiles,
		Noun:           noun,
		Header:         fmt.Sprintf("  %-*s%-*s%s", colName, "NAME", colSize, "SIZE", "META"),
		AddScreen:      nav.FileUpload,
		ConflictScreen: nav.FileSync,
		Fetch: paginatedlist.Fetcher(
			func(cursor string, limit int) ([]repository.FileRow, error) {
				return repo.List(context.Background(), cursor, limit)
			},
			func(row repository.FileRow) (clientmodel.File, error) { return decodeFile(vlt, row) },
			func(row repository.FileRow) clientmodel.File { return clientmodel.File{ID: row.ID} },
		),
		ID:           func(f clientmodel.File) string { return f.ID },
		Revealable:   func(f clientmodel.File) bool { return f.Meta != (clientmodel.FileMeta{}) },
		RenderItem:   renderItem,
		RenderDetail: renderDetail,
		Remove: func(id string) error {
			return repo.Delete(context.Background(), id)
		},
		NewEdit: func(f clientmodel.File) tea.Model {
			return fileedit.New(fileedit.Prop{Vault: vlt, Repo: repo, File: f})
		},
		NewDownload: func(f clientmodel.File) tea.Model {
			return filedownload.New(filedownload.Prop{Vault: vlt, Client: client, File: f})
		},
	})
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
