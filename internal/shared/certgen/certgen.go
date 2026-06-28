// Package certgen creates self-signed TLS material for GophKeeper: one
// certificate that is its own CA, presented by the server and pinned by the
// client as a trusted root.
package certgen

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Options controls the contents of a generated certificate.
type Options struct {
	CommonName string        // subject common name
	Hosts      []string      // Subject Alternative Names; IPs and DNS names are detected automatically
	ValidFor   time.Duration // lifetime from generation time
}

// DefaultOptions returns options suited to a local GophKeeper deployment:
// SANs for localhost and the loopback addresses, valid for ten years.
func DefaultOptions() Options {
	return Options{
		CommonName: "Gophkeeper",
		Hosts:      []string{"localhost", "127.0.0.1", "::1"},
		ValidFor:   10 * 365 * 24 * time.Hour,
	}
}

// Generate returns a self-signed certificate and its private key, both PEM-encoded.
func Generate(opts Options) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: opts.CommonName},
		NotBefore:             now,
		NotAfter:              now.Add(opts.ValidFor),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	for _, h := range opts.Hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM, err = encodePEM("CERTIFICATE", der)
	if err != nil {
		return nil, nil, fmt.Errorf("encode certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM, err = encodePEM("EC PRIVATE KEY", keyDER)
	if err != nil {
		return nil, nil, fmt.Errorf("encode key: %w", err)
	}

	return certPEM, keyPEM, nil
}

// WriteFiles writes the PEM material to disk, the key with owner-only permissions.
func WriteFiles(certPath, keyPath string, certPEM, keyPEM []byte) error {
	if err := writeFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := writeFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	return nil
}

// EnsureFiles creates the cert+key pair only when either file is missing.
func EnsureFiles(certPath, keyPath string, opts Options) error {
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}
	certPEM, keyPEM, err := Generate(opts)
	if err != nil {
		return err
	}
	return WriteFiles(certPath, keyPath, certPEM, keyPEM)
}

// Provision makes the TLS material ready for use: it ensures the cert+key pair
// exists at certPath/keyPath (reissuing it when force is set) and copies the
// public certificate to embedPath for the client to embed.
func Provision(certPath, keyPath, embedPath string, opts Options, force bool) error {
	if err := ensureOrReissue(certPath, keyPath, opts, force); err != nil {
		return err
	}
	return copyCert(certPath, embedPath)
}

func ensureOrReissue(certPath, keyPath string, opts Options, force bool) error {
	if !force {
		return EnsureFiles(certPath, keyPath, opts)
	}
	certPEM, keyPEM, err := Generate(opts)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	return WriteFiles(certPath, keyPath, certPEM, keyPEM)
}

func copyCert(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %q: %w", src, err)
	}
	if err := writeFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("copy cert to %q: %w", dst, err)
	}
	return nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}
	return serial, nil
}

func encodePEM(blockType string, der []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
