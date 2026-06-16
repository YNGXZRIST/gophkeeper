package vault

import (
	"errors"
	"testing"

	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/crypto"
)

func sessionFixture(t *testing.T, password string) (*auth.Session, []byte) {
	t.Helper()
	salt, err := crypto.GenerateBytes(16)
	if err != nil {
		t.Fatalf("salt: %v", err)
	}
	dek, err := crypto.GenerateBytes(32)
	if err != nil {
		t.Fatalf("dek: %v", err)
	}
	wrapper, err := crypto.NewEncryptor(crypto.DeriveKey(password, salt))
	if err != nil {
		t.Fatalf("wrapper: %v", err)
	}
	wrapped, err := wrapper.Encrypt(dek)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	return auth.NewSession("alice", auth.Token{}, auth.Token{}, salt, wrapped), dek
}

func TestUnlockAndRoundTrip(t *testing.T) {
	s, _ := sessionFixture(t, "correct")
	v := New()
	if !v.Locked() {
		t.Fatal("new vault must be locked")
	}
	if err := v.Unlock("correct", s); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if v.Locked() {
		t.Fatal("vault must be unlocked")
	}
	ct, err := v.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := v.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != "secret" {
		t.Fatalf("round-trip = %q, want secret", pt)
	}
}

func TestUnlockWrongPassword(t *testing.T) {
	s, _ := sessionFixture(t, "correct")
	v := New()
	if err := v.Unlock("wrong", s); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("err = %v, want ErrWrongPassword", err)
	}
	if !v.Locked() {
		t.Fatal("vault must stay locked after wrong password")
	}
}

func TestEncryptDecryptLocked(t *testing.T) {
	v := New()
	if _, err := v.Encrypt([]byte("x")); !errors.Is(err, ErrLocked) {
		t.Fatalf("encrypt err = %v, want ErrLocked", err)
	}
	if _, err := v.Decrypt([]byte("x")); !errors.Is(err, ErrLocked) {
		t.Fatalf("decrypt err = %v, want ErrLocked", err)
	}
}

func TestUseDEKAndLock(t *testing.T) {
	dek, _ := crypto.GenerateBytes(32)
	v := New()
	if err := v.UseDEK(dek); err != nil {
		t.Fatalf("use dek: %v", err)
	}
	if v.Locked() {
		t.Fatal("vault must be unlocked after UseDEK")
	}
	v.Lock()
	if !v.Locked() {
		t.Fatal("vault must be locked after Lock")
	}
}
