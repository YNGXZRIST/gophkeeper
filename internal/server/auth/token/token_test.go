package token

import (
	"testing"
	"time"
)

func TestIssuerIssueParse(t *testing.T) {
	tests := []struct {
		name        string
		issuer      *Issuer
		parseSecret []byte
		userID      string
		wantErr     bool
	}{
		{"round trip", NewIssuer([]byte("secret"), time.Hour), []byte("secret"), "user-1", false},
		{"wrong secret", NewIssuer([]byte("secret"), time.Hour), []byte("other"), "user-1", true},
		{"expired", &Issuer{secret: []byte("secret"), ttl: -time.Hour}, []byte("secret"), "user-1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenStr, err := tt.issuer.Issue(tt.userID)
			if err != nil {
				t.Fatalf("Issue() error = %v", err)
			}
			parser := &Issuer{secret: tt.parseSecret, ttl: time.Hour}
			got, err := parser.Parse(tokenStr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Parse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tt.userID {
				t.Errorf("Parse() = %q, want %q", got, tt.userID)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	issuer := NewIssuer([]byte("secret"), time.Hour)
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"not a jwt", "garbage"},
		{"two segments", "aaa.bbb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := issuer.Parse(tt.token); err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.token)
			}
		})
	}
}
