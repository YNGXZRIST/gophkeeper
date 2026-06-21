package crypto

import (
	"bytes"
	"testing"
)

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	key := DeriveKey("master-password", []byte("0123456789abcdef"))
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	return enc
}

func TestNewEncryptor_InvalidKeySize(t *testing.T) {
	if _, err := NewEncryptor([]byte("short")); err == nil {
		t.Fatal("expected error for invalid key size")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	enc := newTestEncryptor(t)

	cases := [][]byte{
		[]byte("hello world"),
		{},
		bytes.Repeat([]byte{0xAB}, 1024),
	}
	for _, plaintext := range cases {
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if bytes.Equal(ciphertext, plaintext) && len(plaintext) > 0 {
			t.Fatal("ciphertext equals plaintext")
		}
		got, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
		}
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	enc := newTestEncryptor(t)
	a, err := enc.Encrypt([]byte("same"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b, err := enc.Encrypt([]byte("same"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	enc := newTestEncryptor(t)
	if _, err := enc.Decrypt([]byte("ab")); err == nil {
		t.Fatal("expected error for ciphertext shorter than nonce")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	enc := newTestEncryptor(t)
	ciphertext, err := enc.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xFF
	if _, err := enc.Decrypt(ciphertext); err == nil {
		t.Fatal("expected authentication failure on tampered ciphertext")
	}
}

func TestGenerateBytes(t *testing.T) {
	b, err := GenerateBytes(16)
	if err != nil {
		t.Fatalf("GenerateBytes: %v", err)
	}
	if len(b) != 16 {
		t.Fatalf("len = %d, want 16", len(b))
	}
	b2, err := GenerateBytes(16)
	if err != nil {
		t.Fatalf("GenerateBytes: %v", err)
	}
	if bytes.Equal(b, b2) {
		t.Fatal("expected different random bytes")
	}
}

func TestGenerateBytes_Zero(t *testing.T) {
	b, err := GenerateBytes(0)
	if err != nil {
		t.Fatalf("GenerateBytes(0): %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("len = %d, want 0", len(b))
	}
}

func TestDeriveKey(t *testing.T) {
	salt := []byte("0123456789abcdef")
	k1 := DeriveKey("password", salt)
	if len(k1) != argonKeyLen {
		t.Fatalf("key len = %d, want %d", len(k1), argonKeyLen)
	}
	k2 := DeriveKey("password", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey not deterministic for same input")
	}
	k3 := DeriveKey("different", salt)
	if bytes.Equal(k1, k3) {
		t.Fatal("different passwords produced same key")
	}
	k4 := DeriveKey("password", []byte("fedcba9876543210"))
	if bytes.Equal(k1, k4) {
		t.Fatal("different salts produced same key")
	}
}
