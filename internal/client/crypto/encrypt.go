package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

// GenerateSalt generating salt transmitted Length
func GenerateSalt(size int) ([]byte, error) {
	salt := make([]byte, size)
	if _, err := rand.Read(salt); err != nil {
		return salt, fmt.Errorf("could not generate salt: %w", err)
	}
	return salt, nil
}

// DeriveKey produces the key-encrypting key that protects the user's data key,
// so the master password itself never has to be stored or transmitted.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}
