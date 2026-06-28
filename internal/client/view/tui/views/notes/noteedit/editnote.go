// Package noteedit is the screen for updating an existing note.
package noteedit

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/notes/internal/noteform"

	tea "charm.land/bubbletea/v2"
)

type Repo interface {
	Update(ctx context.Context, id string, data []byte) error
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
	Note  clientmodel.Note
}

func New(p Prop) tea.Model {
	return noteform.New(p.Vault, "Edit note", p.Note.Data, func(ciphertext []byte) error {
		return p.Repo.Update(context.Background(), p.Note.ID, ciphertext)
	})
}
