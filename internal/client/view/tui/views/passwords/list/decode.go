package list

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"
)

func decodePassword(v *vault.Vault, pb *passwordv1.Password) (clientmodel.Password, error) {
	raw, err := v.Decrypt(pb.GetData())
	if err != nil {
		return clientmodel.Password{}, err
	}
	var data clientmodel.PasswordData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Password{}, err
	}
	return clientmodel.Password{
		ID:        pb.GetId(),
		Data:      data,
		Version:   pb.GetVersion(),
		CreatedAt: pb.GetCreatedAt().AsTime(),
		UpdatedAt: pb.GetUpdatedAt().AsTime(),
	}, nil
}

func maskPassword(s string) string {
	if s == "" {
		return "—"
	}
	return "••••••••"
}
