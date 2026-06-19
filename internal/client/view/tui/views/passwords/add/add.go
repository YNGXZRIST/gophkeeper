// Package add is the screen for adding a password.
package add

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/passwords/internal/passform"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"

	tea "charm.land/bubbletea/v2"
)

type Prop struct {
	Vault  *vault.Vault
	Client passwordv1.PasswordServiceClient
}

func New(p Prop) tea.Model {
	return passform.New(p.Vault, "Password", clientmodel.PasswordData{}, func(ciphertext []byte) error {
		req := &passwordv1.AddRequest{}
		req.SetData(ciphertext)
		_, err := p.Client.Add(context.Background(), req)
		return err
	})
}
