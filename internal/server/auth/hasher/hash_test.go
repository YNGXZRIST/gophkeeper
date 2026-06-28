package hasher

import (
	"testing"
)

func TestHash(t *testing.T) {
	plain := "mySecretPassword123"
	hash, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "" || hash == plain {
		t.Errorf("Hash() returned empty or plain text")
	}
	if len(hash) < 60 {
		t.Errorf("Hash() too short for bcrypt")
	}
}

func TestCompare(t *testing.T) {
	plain := "testPassword"
	hash, err := Hash(plain)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := Compare(hash, plain); err != nil {
		t.Errorf("Compare(correct) = %v", err)
	}
	if err := Compare(hash, "wrong"); err == nil {
		t.Error("Compare(wrong) expected error")
	}
	if err := Compare("not-a-hash", plain); err == nil {
		t.Error("Compare(invalid hash) expected error")
	}
}

func TestHash_UniqueSalt(t *testing.T) {
	plain := "same"
	h1, _ := Hash(plain)
	h2, _ := Hash(plain)
	if h1 == h2 {
		t.Error("Hash() should produce different strings (different salt)")
	}
	if err := Compare(h1, plain); err != nil {
		t.Errorf("Compare(h1) = %v", err)
	}
	if err := Compare(h2, plain); err != nil {
		t.Errorf("Compare(h2) = %v", err)
	}
}
