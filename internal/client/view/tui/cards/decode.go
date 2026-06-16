package cards

import (
	"encoding/json"

	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
)

// decodeCard decrypts a server card's payload and assembles the full client card.
func decodeCard(v *vault.Vault, pb *cardv1.Card) (clientmodel.Card, error) {
	raw, err := v.Decrypt(pb.GetData())
	if err != nil {
		return clientmodel.Card{}, err
	}
	var data clientmodel.CardData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Card{}, err
	}
	return clientmodel.Card{
		ID:        pb.GetId(),
		Data:      data,
		Version:   pb.GetVersion(),
		CreatedAt: pb.GetCreatedAt().AsTime(),
		UpdatedAt: pb.GetUpdatedAt().AsTime(),
	}, nil
}

// maskNumber hides all but the last four digits of a card number.
func maskNumber(s string) string {
	if len(s) < 4 {
		return s
	}
	return "•••• " + s[len(s)-4:]
}
