package vault

import (
	"errors"

	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/crypto"
)

var (
	ErrLocked        = errors.New("vault is locked")
	ErrWrongPassword = errors.New("wrong master password")
)

// Vault holds the unlocked data-encryption key and exposes encrypt/decrypt.
type Vault struct {
	enc *crypto.Encryptor
}

// New returns a locked Vault.
func New() *Vault { return &Vault{} }

// Unlock derives the KEK from the password and unwraps the DEK from the session.
func (v *Vault) Unlock(password string, s *auth.Session) error {
	wrapper, err := crypto.NewEncryptor(crypto.DeriveKey(password, s.EncSalt))
	if err != nil {
		return err
	}
	dek, err := wrapper.Decrypt(s.WrappedDek)
	if err != nil {
		return ErrWrongPassword
	}
	return v.UseDEK(dek)
}

// UseDEK unlocks the vault with an already-available data key (registration path).
func (v *Vault) UseDEK(dek []byte) error {
	enc, err := crypto.NewEncryptor(dek)
	if err != nil {
		return err
	}
	v.enc = enc
	return nil
}

// Encrypt seals plaintext using the vault's data key.
func (v *Vault) Encrypt(b []byte) ([]byte, error) {
	if v.enc == nil {
		return nil, ErrLocked
	}
	return v.enc.Encrypt(b)
}

// Decrypt opens ciphertext using the vault's data key.
func (v *Vault) Decrypt(b []byte) ([]byte, error) {
	if v.enc == nil {
		return nil, ErrLocked
	}
	return v.enc.Decrypt(b)
}

// Locked reports whether the vault has no active data key.
func (v *Vault) Locked() bool { return v.enc == nil }

// Lock clears the data key, returning the vault to a locked state.
func (v *Vault) Lock() { v.enc = nil }
