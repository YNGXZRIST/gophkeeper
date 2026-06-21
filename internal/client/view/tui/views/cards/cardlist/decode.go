package cardlist

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
)

func decodeCard(v *vault.Vault, row repository.CardRow) (clientmodel.Card, error) {
	raw, err := v.Decrypt(row.Data)
	if err != nil {
		return clientmodel.Card{}, err
	}
	var data clientmodel.CardData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Card{}, err
	}
	return clientmodel.Card{
		ID:      row.ID,
		Data:    data,
		Version: row.Version,
	}, nil
}

func maskNumber(s string) string {
	if len(s) < 4 {
		return s
	}
	return "•••• " + s[len(s)-4:]
}
