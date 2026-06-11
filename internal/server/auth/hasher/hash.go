// Package password provides password hash and verification helpers.
package hasher

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is bcrypt hashing cost used by this package.
const DefaultCost = bcrypt.DefaultCost

// Hash hashes plain password using bcrypt.
func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// Compare verifies that plain password matches bcrypt hash.
func Compare(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err != nil {
		return fmt.Errorf("password mismatch: %w", err)
	}
	return nil
}
