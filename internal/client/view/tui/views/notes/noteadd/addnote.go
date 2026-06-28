// Package noteadd is the screen for adding a note.
package noteadd

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/notes/internal/noteform"

	tea "charm.land/bubbletea/v2"
)

type Repo interface {
	Create(ctx context.Context, data []byte) (repository.NoteRow, error)
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
}

func New(p Prop) tea.Model {
	return noteform.New(p.Vault, "Note", clientmodel.NoteData{}, func(ciphertext []byte) error {
		_, err := p.Repo.Create(context.Background(), ciphertext)
		return err
	})
}
