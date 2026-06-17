// Package edit is the screen for updating an existing card.
package edit

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
	Card   clientmodel.Card
}

func New(p Prop) tea.Model {
	return cardform.New(p.Vault, "Edit card", p.Card.Data, func(ciphertext []byte) error {
		req := &cardv1.UpdateRequest{}
		req.SetId(p.Card.ID)
		req.SetData(ciphertext)
		req.SetVersion(p.Card.Version)
		_, err := p.Client.Update(context.Background(), req)
		return err
	})
}
