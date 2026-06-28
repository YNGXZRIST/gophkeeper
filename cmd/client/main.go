package main

import (
	"fmt"
	"gophkeeper/internal/client/app"
	"log"
	"os"
)

// Build-time values injected via -ldflags "-X main.<name>=...".
var (
	grpcServerAddr = "localhost:8080"
	buildVersion   = "N/A"
	buildDate      = "N/A"
)

func main() {
	if versionRequested(os.Args[1:]) {
		fmt.Printf("GophKeeper client\nversion: %s\nbuild date: %s\n", buildVersion, buildDate)
		return
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// versionRequested reports whether the user asked for version information.
func versionRequested(args []string) bool {
	for _, a := range args {
		if a == "-version" || a == "--version" {
			return true
		}
	}
	return false
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
