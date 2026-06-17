// Package add is the screen for adding a card.
package add

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/cards/internal/cardform"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"

	tea "charm.land/bubbletea/v2"
)

type Prop struct {
	Vault  *vault.Vault
	Client cardv1.CardServiceClient
}

func New(p Prop) tea.Model {
	return cardform.New(p.Vault, "Debit card", clientmodel.CardData{}, func(ciphertext []byte) error {
		req := &cardv1.AddRequest{}
		req.SetData(ciphertext)
		_, err := p.Client.Add(context.Background(), req)
		return err
	})
}
