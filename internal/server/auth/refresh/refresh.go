// Package refresh generates and hashes high-entropy refresh tokens.
package refresh

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const tokenBytes = 32

// Generate returns a new high-entropy refresh token in plaintext form.
func Generate() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the deterministic HMAC-SHA256 hash of a refresh token, suitable
// for indexed storage and lookup.
func Hash(plain string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(plain))
	return hex.EncodeToString(mac.Sum(nil))
}

// Equal reports whether two token hashes match in constant time.
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
