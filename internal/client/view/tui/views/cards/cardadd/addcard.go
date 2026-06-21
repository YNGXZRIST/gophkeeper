// Package cardadd is the screen for adding a card.
package cardadd

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/cards/internal/cardform"
	"gophkeeper/internal/client/view/tui/views/home"

	tea "charm.land/bubbletea/v2"
)

type Repo interface {
	Create(ctx context.Context, data []byte) (repository.CardRow, error)
}

type Prop struct {
	Vault *vault.Vault
	Repo  Repo
}

func New(p Prop) tea.Model {
	return cardform.New(p.Vault, home.LabelCards, clientmodel.CardData{}, func(ciphertext []byte) error {
		_, err := p.Repo.Create(context.Background(), ciphertext)
		return err
	})
}
