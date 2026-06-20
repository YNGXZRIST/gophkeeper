// Package edit is the screen for updating an existing note.
package noteedit

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/notes/internal/noteform"
	notev1 "gophkeeper/internal/shared/proto/note/v1"

	tea "charm.land/bubbletea/v2"
)

type Prop struct {
	Vault  *vault.Vault
	Client notev1.NoteServiceClient
	Note   clientmodel.Note
}

func New(p Prop) tea.Model {
	return noteform.New(p.Vault, "Edit note", p.Note.Data, func(ciphertext []byte) error {
		req := &notev1.UpdateRequest{}
		req.SetId(p.Note.ID)
		req.SetData(ciphertext)
		req.SetVersion(p.Note.Version)
		_, err := p.Client.Update(context.Background(), req)
		return err
	})
}
