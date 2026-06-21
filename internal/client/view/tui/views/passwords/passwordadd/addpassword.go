// Package passwordadd is the screen for adding a password.
package passwordadd

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/passwords/internal/passform"

	tea "charm.land/bubbletea/v2"
)

type Repo interface {
	Create(ctx context.Context, data []byte) (repository.PasswordRow, error)
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
}

func New(p Prop) tea.Model {
	return passform.New(p.Vault, "Password", clientmodel.PasswordData{}, func(ciphertext []byte) error {
		_, err := p.Repo.Create(context.Background(), ciphertext)
		return err
	})
}
