// Package passwordedit is the screen for updating an existing password.
package passwordedit

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/passwords/internal/passform"

	tea "charm.land/bubbletea/v2"
)

type Repo interface {
	Update(ctx context.Context, id string, data []byte) error
}

type Prop struct {
	Vault    *vault.Vault
	Repo     Repo
	Password clientmodel.Password
}

func New(p Prop) tea.Model {
	return passform.New(p.Vault, "Edit password", p.Password.Data, func(ciphertext []byte) error {
		return p.Repo.Update(context.Background(), p.Password.ID, ciphertext)
	})
}
