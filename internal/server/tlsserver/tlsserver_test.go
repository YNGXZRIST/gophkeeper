package tlsserver

import (
	"os"
	"path/filepath"
	"testing"

	"gophkeeper/internal/shared/certgen"
)

func writePair(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()
	certPath = filepath.Join(dir, "server.crt")
	keyPath = filepath.Join(dir, "server.key")
	certPEM, keyPEM, err := certgen.Generate(certgen.DefaultOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := certgen.WriteFiles(certPath, keyPath, certPEM, keyPEM); err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	return certPath, keyPath
}

func TestLoadCredentialsOK(t *testing.T) {
	certPath, keyPath := writePair(t)
	creds, err := LoadCredentials(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if creds == nil {
		t.Fatalf("LoadCredentials returned nil credentials")
	}
}

func TestLoadCredentialsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadCredentials(
		filepath.Join(dir, "absent.crt"),
		filepath.Join(dir, "absent.key"),
	)
	if err == nil {
		t.Fatalf("LoadCredentials: expected error for missing files")
	}
}

func TestLoadCredentialsGarbage(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, []byte("not pem"), 0o644); err != nil {
		t.Fatalf("write garbage cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("not pem"), 0o600); err != nil {
		t.Fatalf("write garbage key: %v", err)
	}
	if _, err := LoadCredentials(certPath, keyPath); err == nil {
		t.Fatalf("LoadCredentials: expected error for garbage files")
	}
}
