// Package add is the screen for adding a note.
package noteadd

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
}

func New(p Prop) tea.Model {
	return noteform.New(p.Vault, "Note", clientmodel.NoteData{}, func(ciphertext []byte) error {
		req := &notev1.AddRequest{}
		req.SetData(ciphertext)
		_, err := p.Client.Add(context.Background(), req)
		return err
	})
}
