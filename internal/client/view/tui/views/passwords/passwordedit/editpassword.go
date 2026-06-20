// Package edit is the screen for updating an existing password.
package passwordedit

import (
	"context"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/views/passwords/internal/passform"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"

	tea "charm.land/bubbletea/v2"
)

type Prop struct {
	Vault    *vault.Vault
	Client   passwordv1.PasswordServiceClient
	Password clientmodel.Password
}

func New(p Prop) tea.Model {
	return passform.New(p.Vault, "Edit password", p.Password.Data, func(ciphertext []byte) error {
		req := &passwordv1.UpdateRequest{}
		req.SetId(p.Password.ID)
		req.SetData(ciphertext)
		req.SetVersion(p.Password.Version)
		_, err := p.Client.Update(context.Background(), req)
		return err
	})
}
