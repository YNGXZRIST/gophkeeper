package main

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/app"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()
	a, err := run(os.Args[1:])
	if err != nil {
		log.Fatalf("fatal error: %v", err)
	}

	<-ctx.Done()

	log.Println("shutting down application...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if errShutdown := a.Shutdown(ctxShutdown); errShutdown != nil {
		log.Printf("shutdown error: %v", errShutdown)
	}

	log.Println("application stopped")
}
func run(args []string) (*app.App, error) {
	a, err := app.Bootstrap(args)
	if err != nil {
		return nil, fmt.Errorf("fatal error: %v", err)
	}
	return a, nil

}
