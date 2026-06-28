// Package app wires the client dependencies together and runs the TUI program.
package app

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
	"gophkeeper/internal/client/tlsclient"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/root"
	"gophkeeper/internal/shared/logger"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
	notev1 "gophkeeper/internal/shared/proto/note/v1"
	passwordv1 "gophkeeper/internal/shared/proto/password/v1"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	mg "gophkeeper/migrations/client"
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
)

type App struct {
	program *tea.Program
	grpc    *grpc.ClientConn
	db      *sql.DB
}

type repos struct {
	notes     *repository.EntryRepo
	passwords *repository.EntryRepo
	cards     *repository.EntryRepo
	files     *repository.FilesRepo
	session   *repository.SessionRepo
	syncState *repository.SyncStateRepo
}

type clients struct {
	user     userv1.UserServiceClient
	card     cardv1.CardServiceClient
	password passwordv1.PasswordServiceClient
	note     notev1.NoteServiceClient
	file     filev1.FileServiceClient
}

func Bootstrap(addr string) (*App, error) {
	dbConn, err := db.Open()
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err = mg.Migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	lg, err := logger.Initialize(&logger.Config{
		Mode:   logger.ModeProduction,
		Dir:    logger.DefaultLogDir,
		Prefix: "client",
	})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	r := buildRepos(dbConn)
	if err = checkSession(r.session, lg); err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}

	grpcConn, err := dialGRPC(addr, r.session, lg)
	if err != nil {
		return nil, fmt.Errorf("dial grpc: %w", err)
	}

	c := buildClients(grpcConn)
	pool := buildPool(r, c)
	program := tea.NewProgram(root.New(buildDeps(r, c, pool, lg)))

	return &App{program: program, grpc: grpcConn, db: dbConn}, nil
}

func (a *App) Run() error {
	_, err := a.program.Run()
	return err
}

func (a *App) Shutdown() error {
	return errors.Join(
		a.grpc.Close(),
		a.db.Close(),
	)
}

func buildRepos(dbConn *sql.DB) repos {
	return repos{
		notes:     repository.NewEntryRepo(dbConn, repository.TableNote),
		passwords: repository.NewEntryRepo(dbConn, repository.TablePassword),
		cards:     repository.NewEntryRepo(dbConn, repository.TableCard),
		files:     repository.NewFilesRepo(dbConn),
		session:   repository.NewSessionRepo(dbConn),
		syncState: repository.NewSyncStateRepo(dbConn),
	}
}

func buildClients(conn *grpc.ClientConn) clients {
	return clients{
		user:     userv1.NewUserServiceClient(conn),
		card:     cardv1.NewCardServiceClient(conn),
		password: passwordv1.NewPasswordServiceClient(conn),
		note:     notev1.NewNoteServiceClient(conn),
		file:     filev1.NewFileServiceClient(conn),
	}
}

func buildPool(r repos, c clients) *syncclient.Pool {
	notesSyncer := syncer.New(syncnotes.NewRepo(r.notes, r.syncState), syncnotes.NewClient(c.note))
	cardsSyncer := syncer.New(synccards.NewRepo(r.cards, r.syncState), synccards.NewClient(c.card))
	passwordsSyncer := syncer.New(syncpasswords.NewRepo(r.passwords, r.syncState), syncpasswords.NewClient(c.password))
	filesSyncer := syncer.New(syncfiles.NewRepo(r.files, r.syncState), syncfiles.NewClient(c.file))
	return syncclient.New(notesSyncer, cardsSyncer, passwordsSyncer, filesSyncer)
}

func buildDeps(r repos, c clients, pool *syncclient.Pool, lg *slog.Logger) root.Deps {
	return root.Deps{
		UserClient:    c.user,
		NotesRepo:     r.notes,
		CardsRepo:     r.cards,
		PasswordsRepo: r.passwords,
		FilesRepo:     r.files,
		FileClient:    c.file,
		Sync:          pool,
		Logger:        lg,
		SessionStore:  r.session,
		Vault:         vault.New(),
	}
}

func dialGRPC(addr string, sessions *repository.SessionRepo, lg *slog.Logger) (*grpc.ClientConn, error) {
	creds, err := tlsclient.Credentials()
	if err != nil {
		return nil, fmt.Errorf("tls credentials: %w", err)
	}
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithChainUnaryInterceptor(
			interceptor.UnaryRefreshInterceptor(sessions, lg),
			interceptor.UnaryAuthInterceptor(sessions),
		),
		grpc.WithChainStreamInterceptor(
			interceptor.StreamRefreshInterceptor(sessions, lg),
			interceptor.StreamAuthInterceptor(sessions),
		),
	)
}

func checkSession(sessions *repository.SessionRepo, lg *slog.Logger) error {
	_, err := sessions.Get(context.Background())
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, repository.ErrNoSession) {
		return nil
	}
	if parseErr, ok := errors.AsType[*repository.ErrParseToken](err); ok {
		lg.Error("parse stored session", slog.Any("error", parseErr))
		return nil
	}
	return err
}
