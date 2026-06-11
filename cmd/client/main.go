package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/db"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/view/tui/root"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	mg "gophkeeper/migrations/client"
	"log"
	"os"

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
		grpc.WithUnaryInterceptor(auth.UnaryRefreshInterceptor(sessionRepo)),
		grpc.WithUnaryInterceptor(auth.UnaryAuthInterceptor(sessionRepo)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = grpcConn.Close() }()
	userClient := userv1.NewUserServiceClient(grpcConn)

	if _, err = tea.NewProgram(root.New(root.Deps{Client: userClient, SessionsStore: sessionRepo})).Run(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
	_ = session

}
