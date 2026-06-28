package auth

import (
	"bytes"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestNewToken_Valid(t *testing.T) {
	exp := time.Now().Add(time.Hour).Truncate(time.Second)
	raw := signedToken(t, exp)

	tok, err := NewToken(raw)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if tok.Raw != raw {
		t.Fatalf("Raw = %q, want %q", tok.Raw, raw)
	}
	if !tok.ExpiresAt.Equal(exp) {
		t.Fatalf("ExpiresAt = %v, want %v", tok.ExpiresAt, exp)
	}
}

func TestNewToken_Invalid(t *testing.T) {
	if _, err := NewToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestToken_Expired(t *testing.T) {
	tests := []struct {
		name    string
		expires time.Time
		skew    time.Duration
		want    bool
	}{
		{"future not expired", time.Now().Add(time.Hour), 0, false},
		{"past expired", time.Now().Add(-time.Hour), 0, true},
		{"future within skew is expired", time.Now().Add(time.Minute), 5 * time.Minute, true},
		{"future beyond skew not expired", time.Now().Add(time.Hour), 5 * time.Minute, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := Token{ExpiresAt: tt.expires}
			if got := tok.Expired(tt.skew); got != tt.want {
				t.Fatalf("Expired(%v) = %v, want %v", tt.skew, got, tt.want)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	access := Token{Raw: "a"}
	refresh := Token{Raw: "r"}
	salt := []byte("salt")
	dek := []byte("dek")

	s := NewSession("user", access, refresh, salt, dek)
	if s.Login != "user" {
		t.Fatalf("Login = %q", s.Login)
	}
	if s.Access.Raw != "a" || s.Refresh.Raw != "r" {
		t.Fatalf("tokens not set: %+v", s)
	}
	if !bytes.Equal(s.EncSalt, salt) || !bytes.Equal(s.WrappedDek, dek) {
		t.Fatalf("keys not set: %+v", s)
	}
}

func TestCredentials_Fields(t *testing.T) {
	c := Credentials{
		Login:        "user",
		AccessToken:  "at",
		RefreshToken: "rt",
		EncSalt:      []byte("s"),
		WrappedDek:   []byte("d"),
	}
	if c.Login != "user" || c.AccessToken != "at" || c.RefreshToken != "rt" {
		t.Fatalf("unexpected credentials: %+v", c)
	}
}
