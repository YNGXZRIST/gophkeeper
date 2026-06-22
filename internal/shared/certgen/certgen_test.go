package certgen

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func parseCert(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("decode cert PEM: no block found")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

func TestGenerateDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	certPEM, keyPEM, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	cert := parseCert(t, certPEM)
	if !cert.IsCA {
		t.Errorf("expected certificate to be a CA")
	}
	if cert.Subject.CommonName != opts.CommonName {
		t.Errorf("CommonName = %q, want %q", cert.Subject.CommonName, opts.CommonName)
	}

	hasDNS := false
	for _, n := range cert.DNSNames {
		if n == "localhost" {
			hasDNS = true
		}
	}
	if !hasDNS {
		t.Errorf("DNSNames %v missing localhost", cert.DNSNames)
	}

	wantIPs := []string{"127.0.0.1", "::1"}
	for _, want := range wantIPs {
		found := false
		for _, ip := range cert.IPAddresses {
			if ip.Equal(net.ParseIP(want)) {
				found = true
			}
		}
		if !found {
			t.Errorf("IPAddresses %v missing %s", cert.IPAddresses, want)
		}
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		t.Fatalf("decode key PEM: no block found")
	}
	if _, err := x509.ParseECPrivateKey(block.Bytes); err != nil {
		t.Fatalf("parse EC private key: %v", err)
	}
}

func TestGenerateCustomHosts(t *testing.T) {
	opts := Options{
		CommonName: "custom",
		Hosts:      []string{"example.com", "10.0.0.1", "api.local"},
		ValidFor:   time.Hour,
	}
	certPEM, _, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	cert := parseCert(t, certPEM)

	if len(cert.DNSNames) != 2 {
		t.Errorf("DNSNames = %v, want 2 entries", cert.DNSNames)
	}
	if len(cert.IPAddresses) != 1 {
		t.Errorf("IPAddresses = %v, want 1 entry", cert.IPAddresses)
	}
	if !cert.IPAddresses[0].Equal(net.ParseIP("10.0.0.1")) {
		t.Errorf("IPAddresses[0] = %v, want 10.0.0.1", cert.IPAddresses[0])
	}
}

func TestWriteFilesAndPermissions(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	certPEM, keyPEM, err := Generate(DefaultOptions())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := WriteFiles(certPath, keyPath, certPEM, keyPEM); err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	gotCert, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	if !bytes.Equal(gotCert, certPEM) {
		t.Errorf("written cert content differs from input")
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("key permissions = %o, want 600", perm)
	}
}

func TestWriteFilesErrorParentIsFile(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file, then try to use it as a parent directory.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	certPath := filepath.Join(blocker, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	err := WriteFiles(certPath, keyPath, []byte("cert"), []byte("key"))
	if err == nil {
		t.Fatalf("WriteFiles: expected error when parent is a file")
	}
}

func TestEnsureFilesCreatesAndDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	if err := EnsureFiles(certPath, keyPath, DefaultOptions()); err != nil {
		t.Fatalf("EnsureFiles (create): %v", err)
	}
	first, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	firstSerial := parseCert(t, first).SerialNumber

	// Second call must not overwrite the existing pair.
	if err := EnsureFiles(certPath, keyPath, DefaultOptions()); err != nil {
		t.Fatalf("EnsureFiles (noop): %v", err)
	}
	second, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("EnsureFiles overwrote existing certificate")
	}
	if parseCert(t, second).SerialNumber.Cmp(firstSerial) != 0 {
		t.Errorf("serial changed across EnsureFiles calls")
	}
}

func TestEnsureFilesGenerateErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	// Missing cert+key so Generate runs, but ValidFor is fine; instead force a
	// write failure: parent of cert is a file.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	certPath := filepath.Join(blocker, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	if err := EnsureFiles(certPath, keyPath, DefaultOptions()); err == nil {
		t.Fatalf("EnsureFiles: expected error on unwritable cert path")
	}
}

func TestProvisionWritesPairAndEmbedCopy(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "certs", "server.crt")
	keyPath := filepath.Join(dir, "certs", "server.key")
	embedPath := filepath.Join(dir, "embed", "server.crt")

	if err := Provision(certPath, keyPath, embedPath, DefaultOptions(), false); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	cert, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	embed, err := os.ReadFile(embedPath)
	if err != nil {
		t.Fatalf("read embed: %v", err)
	}
	if !bytes.Equal(cert, embed) {
		t.Errorf("embed copy differs from cert")
	}
}

func TestProvisionForceReissues(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	embedPath := filepath.Join(dir, "embed.crt")

	if err := Provision(certPath, keyPath, embedPath, DefaultOptions(), false); err != nil {
		t.Fatalf("Provision (initial): %v", err)
	}
	first, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	firstSerial := parseCert(t, first).SerialNumber

	if err := Provision(certPath, keyPath, embedPath, DefaultOptions(), true); err != nil {
		t.Fatalf("Provision (force): %v", err)
	}
	second, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert after force: %v", err)
	}
	if parseCert(t, second).SerialNumber.Cmp(firstSerial) == 0 {
		t.Errorf("force did not reissue: serial unchanged")
	}

	embed, err := os.ReadFile(embedPath)
	if err != nil {
		t.Fatalf("read embed: %v", err)
	}
	if !bytes.Equal(second, embed) {
		t.Errorf("embed copy not updated after force reissue")
	}
}

func TestProvisionForceWriteError(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	certPath := filepath.Join(blocker, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	embedPath := filepath.Join(dir, "embed.crt")

	if err := Provision(certPath, keyPath, embedPath, DefaultOptions(), true); err == nil {
		t.Fatalf("Provision: expected error on unwritable cert path")
	}
}

func TestProvisionEmbedCopyError(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	// embed parent is a file -> copy step must fail.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup blocker: %v", err)
	}
	embedPath := filepath.Join(blocker, "server.crt")

	if err := Provision(certPath, keyPath, embedPath, DefaultOptions(), false); err == nil {
		t.Fatalf("Provision: expected error copying embed cert")
	}
}
