package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophkeeper/internal/client/db"
	"gophkeeper/internal/client/interceptor"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/sync/synccards"
	"gophkeeper/internal/client/sync/syncclient"
	"gophkeeper/internal/client/sync/syncer"
	"gophkeeper/internal/client/sync/syncfiles"
	"gophkeeper/internal/client/sync/syncnotes"
	"gophkeeper/internal/client/sync/syncpasswords"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/root"
	"gophkeeper/internal/shared/logger"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	notev1 "gophkeeper/internal/shared/proto/note/v1"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"
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
	notesRepo := repository.NewNotesRepo(conn)
	passwordsRepo := repository.NewPasswordsRepo(conn)
	cardsRepo := repository.NewCardsRepo(conn)
	filesRepo := repository.NewFilesRepo(conn)
	sessionRepo := repository.NewSessionRepo(conn)
	_, err = sessionRepo.Get(context.Background())
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
		grpc.WithChainStreamInterceptor(
			interceptor.StreamRefreshInterceptor(sessionRepo, lg),
			interceptor.StreamAuthInterceptor(sessionRepo),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = grpcConn.Close() }()
	userClient := userv1.NewUserServiceClient(grpcConn)
	cardClient := cardv1.NewCardServiceClient(grpcConn)
	passwordClient := passwordv1.NewPasswordServiceClient(grpcConn)
	noteClient := notev1.NewNoteServiceClient(grpcConn)
	fileClient := filev1.NewFileServiceClient(grpcConn)
	syncStateRepo := repository.NewSyncStateRepo(conn)
	notesSyncer := syncer.New(syncnotes.NewRepo(notesRepo, syncStateRepo), syncnotes.NewClient(noteClient))
	cardsSyncer := syncer.New(synccards.NewRepo(cardsRepo, syncStateRepo), synccards.NewClient(cardClient))
	passwordsSyncer := syncer.New(syncpasswords.NewRepo(passwordsRepo, syncStateRepo), syncpasswords.NewClient(passwordClient))
	filesSyncer := syncer.New(syncfiles.NewRepo(filesRepo, syncStateRepo), syncfiles.NewClient(fileClient))
	pool := syncclient.New(notesSyncer, cardsSyncer, passwordsSyncer, filesSyncer)
	if _, err = tea.NewProgram(root.New(root.Deps{
		UserClient:    userClient,
		NotesRepo:     notesRepo,
		CardsRepo:     cardsRepo,
		FileClient:    fileClient,
		FilesRepo:     filesRepo,
		PasswordsRepo: passwordsRepo,
		Sync:          pool,
		Logger:        lg,
		SessionStore:  sessionRepo,
		Vault:         vault.New()})).Run(); err != nil {
		log.Fatal("could not start program:\n", err)
	}

}
