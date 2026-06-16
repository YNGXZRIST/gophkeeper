package crypto

import (
	"crypto/aes"
	"crypto/cipher"
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

type Encryptor struct {
	aead cipher.AEAD
}

// NewEncryptor builds an AES-GCM encryptor over the given key.
func NewEncryptor(key []byte) (*Encryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return &Encryptor{aead: gcm}, nil
}

// Encrypt seals plaintext and returns nonce||ciphertext||tag.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce, err := GenerateBytes(e.aead.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens a nonce||ciphertext||tag blob produced by Encrypt.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, payload := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// GenerateBytes generating salt transmitted Length
func GenerateBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return bytes, fmt.Errorf("could not generate salt: %w", err)
	}
	return bytes, nil
}

// DeriveKey produces the key-encrypting key that protects the user's data key,
// so the master password itself never has to be stored or transmitted.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}
