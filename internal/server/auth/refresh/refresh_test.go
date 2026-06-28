package refresh

import (
	"encoding/base64"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"first"},
		{"second"},
		{"third"},
	}
	seen := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := Generate()
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			raw, err := base64.RawURLEncoding.DecodeString(tok)
			if err != nil {
				t.Fatalf("decode error = %v", err)
			}
			if len(raw) != tokenBytes {
				t.Errorf("len = %d, want %d", len(raw), tokenBytes)
			}
			if seen[tok] {
				t.Errorf("duplicate token %q", tok)
			}
			seen[tok] = true
		})
	}
}

func TestHash(t *testing.T) {
	secret := []byte("server-secret")
	tests := []struct {
		name      string
		x         string
		xSecret   []byte
		y         string
		ySecret   []byte
		wantEqual bool
	}{
		{"same input same secret", "tok", secret, "tok", secret, true},
		{"different input", "tok-a", secret, "tok-b", secret, false},
		{"different secret", "tok", secret, "tok", []byte("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Hash(tt.x, tt.xSecret) == Hash(tt.y, tt.ySecret)
			if got != tt.wantEqual {
				t.Errorf("equality = %v, want %v", got, tt.wantEqual)
			}
			if h := Hash(tt.x, tt.xSecret); len(h) != 64 {
				t.Errorf("hash len = %d, want 64", len(h))
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal", "abc", "abc", true},
		{"different same length", "abc", "abd", false},
		{"different length", "abc", "abcd", false},
		{"both empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.a, tt.b); got != tt.want {
				t.Errorf("Equal(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
