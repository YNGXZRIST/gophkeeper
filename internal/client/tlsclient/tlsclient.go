// Package tlsclient provides gRPC transport credentials that trust the embedded
// GophKeeper server certificate.
package tlsclient

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"

	"google.golang.org/grpc/credentials"
)

//go:embed cert/server.crt
var serverCert []byte

// Credentials returns gRPC transport credentials pinning the embedded server certificate.
func Credentials() (credentials.TransportCredentials, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(serverCert) {
		return nil, fmt.Errorf("parse embedded certificate")
	}
	return credentials.NewTLS(&tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}), nil
}
