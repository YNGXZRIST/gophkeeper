// Package cardedit is the screen for updating an existing card.
package cardedit

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/cards/internal/cardform"

	tea "charm.land/bubbletea/v2"
)

const label = "Edit Card"

type Repo interface {
	Update(ctx context.Context, id string, data []byte) error
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
	Card  clientmodel.Card
}

func New(p Prop) tea.Model {
	return cardform.New(p.Vault, label, p.Card.Data, func(ciphertext []byte) error {
		return p.Repo.Update(context.Background(), p.Card.ID, ciphertext)
	})
}
