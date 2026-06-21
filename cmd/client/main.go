package main

import (
	"fmt"
	"gophkeeper/internal/client/app"
	"log"
)

// grpcServerAddr is injected at build time via -ldflags "-X main.grpcServerAddr=host:port".
var grpcServerAddr = "localhost:8080"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run owns the application lifecycle so that deferred cleanup runs before the
// process exits; main only translates a returned error into a non-zero exit.
func run() error {
	application, err := app.Bootstrap(grpcServerAddr)
	if err != nil {
		return err
	}
	defer func() { _ = application.Shutdown() }()

	if err := application.Run(); err != nil {
		return fmt.Errorf("could not start program: %w", err)
	}
	return nil
}
