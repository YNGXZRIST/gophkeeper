// Package tlsserver loads the server's TLS credentials from disk.
package tlsserver

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc/credentials"
)

// LoadCredentials reads the cert+key pair and returns gRPC transport credentials.
func LoadCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}), nil
}
