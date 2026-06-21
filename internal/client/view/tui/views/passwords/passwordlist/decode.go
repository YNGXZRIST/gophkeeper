package passwordlist

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
)

func decodePassword(v *vault.Vault, row repository.PasswordRow) (clientmodel.Password, error) {
	raw, err := v.Decrypt(row.Data)
	if err != nil {
		return clientmodel.Password{}, err
	}
	var data clientmodel.PasswordData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Password{}, err
	}
	return clientmodel.Password{
		ID:      row.ID,
		Data:    data,
		Version: row.Version,
	}, nil
}

func maskPassword(s string) string {
	if s == "" {
		return "—"
	}
	return "••••••••"
}
