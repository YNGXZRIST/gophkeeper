package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/client/db"
	"gophkeeper/internal/client/interceptor"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/root"
	"gophkeeper/internal/shared/logger"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	mg "gophkeeper/migrations/client"
	"log"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// grpcServerAddr is the address of the remote gRPC server. It is injected at
// build time via -ldflags "-X main.grpcServerAddr=host:port"; the default
// targets a locally running server for development.
var grpcServerAddr = "localhost:8080"

func main() {
	//TODO: move to app
	conn, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	_ = conn
	err = mg.Migrate()
	if err != nil {
		log.Fatal(err)
	}
	lg, err := logger.Initialize(&logger.Config{
		Mode:   logger.ModeProduction,
		Dir:    logger.DefaultLogDir,
		Prefix: "client",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = lg.Sync() }()

	sessionRepo := repository.NewSessionRepo(conn)
	session, err := sessionRepo.Get(context.Background())
	if err != nil {
		switch true {
		case errors.Is(err, sql.ErrNoRows):
			// continue
			break
		case errors.Is(err, repository.ErrNoSession):
			//continue
			break
		default:
			err, ok := errors.AsType[*repository.ErrParseToken](err)
			if !ok {
				log.Fatal(err)
			}
			fmt.Printf("error:%+v\n", err)
		}
	}
	grpcConn, err := grpc.NewClient(
		grpcServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			interceptor.UnaryRefreshInterceptor(sessionRepo, lg),
			interceptor.UnaryAuthInterceptor(sessionRepo),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = grpcConn.Close() }()
	userClient := userv1.NewUserServiceClient(grpcConn)
	cardClient := cardv1.NewCardServiceClient(grpcConn)

	if _, err = tea.NewProgram(root.New(root.Deps{UserClient: userClient, CardClient: cardClient, SessionStore: sessionRepo, Vault: vault.New()})).Run(); err != nil {
		log.Fatal("could not start program:\n", err)
	}
	_ = session

}
