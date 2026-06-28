// Command certgen creates the self-signed TLS pair used by GophKeeper.
//
// It writes the cert+key pair the server reads at runtime and copies the public
// certificate into the client package so it can be embedded at build time. By
// default an existing pair is reused; pass -force to reissue it.
package main

import (
	"gophkeeper/internal/shared/certgen"
	"gophkeeper/internal/shared/certgen/config"
	"log"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	cfg, err := config.Parse(args)
	if err != nil {
		return err
	}

	opts := certgen.DefaultOptions()
	opts.Hosts = cfg.Hosts

	if err := certgen.Provision(cfg.CertPath, cfg.KeyPath, cfg.EmbedPath, opts, cfg.Force); err != nil {
		return err
	}

	log.Printf("certs ready: %s, %s (embedded copy: %s)", cfg.CertPath, cfg.KeyPath, cfg.EmbedPath)
	return nil
}
