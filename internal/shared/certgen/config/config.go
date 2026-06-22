// Package config parses command-line flags for the certgen tool into a Config.
package config

import (
	"flag"
	"strings"
)

// Defaults for the certgen invocation: the server reads cert+key from ./certs,
// the client embeds a copy of the public certificate from its package tree.
const (
	DefaultCertPath  = "./certs/server.crt"
	DefaultKeyPath   = "./certs/server.key"
	DefaultEmbedPath = "./internal/client/tlsclient/cert/server.crt"
	DefaultHosts     = "localhost,127.0.0.1,::1"
)

// Config holds the resolved parameters of a certgen run.
type Config struct {
	CertPath  string
	KeyPath   string
	EmbedPath string
	Hosts     []string
	Force     bool
}

// Parse turns command-line arguments into a Config, applying defaults for any
// flag the caller did not pass.
func Parse(args []string) (*Config, error) {
	var (
		certPath  string
		keyPath   string
		embedPath string
		hosts     string
		force     bool
	)
	fs := flag.NewFlagSet("certgen", flag.ContinueOnError)
	fs.StringVar(&certPath, "cert", DefaultCertPath, "path to the server certificate")
	fs.StringVar(&keyPath, "key", DefaultKeyPath, "path to the server private key")
	fs.StringVar(&embedPath, "embed", DefaultEmbedPath, "path the public cert is copied to for client embedding")
	fs.StringVar(&hosts, "host", DefaultHosts, "comma-separated SAN hosts (DNS names or IPs)")
	fs.BoolVar(&force, "force", false, "reissue the pair even if it already exists")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Config{
		CertPath:  certPath,
		KeyPath:   keyPath,
		EmbedPath: embedPath,
		Hosts:     splitHosts(hosts),
		Force:     force,
	}, nil
}

func splitHosts(hosts string) []string {
	parts := strings.Split(hosts, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
