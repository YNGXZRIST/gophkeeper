package app

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/auth/token"
	"gophkeeper/internal/server/config"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/server/repository"
	"gophkeeper/internal/server/service"
	"gophkeeper/internal/server/transport"
	"gophkeeper/internal/shared/errors/labelerrors"
	"gophkeeper/internal/shared/logger"
	mg "gophkeeper/migrations/server"
)

const (
	labelApp = "APP"
)

type Options struct {
}
type dbCloser interface {
	Close() error
}
type App struct {
	db     dbCloser
	server transport.Server
}

func Bootstrap(args []string) (_ *App, err error) {
	opt, err := config.Load(args)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	log, err := logger.Initialize(&logger.Config{
		Mode:    opt.AppMode,
		Dir:     opt.LogDir,
		Prefix:  "server",
		Console: true,
	})
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	dbConn, err := initDB(opt)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}
	defer func() {
		if err != nil {
			_ = dbConn.Close()
		}
	}()
	err = mg.Migrate(opt.DSN)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	mgr := trmanager.NewManager(dbConn)
	repos := buildRepos(dbConn)
	refreshIssuer := repository.NewRefreshIssuer(repos.RefreshToken, []byte(opt.RefreshSecret), repository.DefaultRefreshTTL)
	authWriter := repository.NewAuthWriter(mgr, repos, refreshIssuer)
	issuer := token.NewIssuer([]byte(opt.JWTSecret), token.DefaultAccessTTL)
	services := buildServices(serviceDeps{
		Repos:   repos,
		Auth:    authWriter,
		Issuer:  issuer,
		Manager: mgr,
	})

	server, err := transport.NewServer(
		transport.ServerProp{
			Config:      &transport.Config{Transport: opt.Transport, Address: opt.Address},
			Services:    services,
			Logger:      log,
			TokenParser: issuer,
		})
	if err != nil {
		return nil, fmt.Errorf("new server: %w", err)
	}

	return &App{
		db:     dbConn,
		server: server,
	}, nil
}
func (a *App) Run() error {
	return a.server.Start()
}
func (a *App) Shutdown(ctx context.Context) error {
	err := a.server.Stop(ctx)
	if dbErr := a.db.Close(); dbErr != nil && err == nil {
		err = dbErr
	}
	return err
}
func initDB(cfg *config.Flags) (*conn.DB, error) {
	dbConfig := conn.NewCfg(cfg.DSN)
	newConn, err := conn.NewConn(dbConfig)
	if err != nil {
		return nil, labelerrors.NewLabelError(labelApp+".InitDB", fmt.Errorf("error creating database connection: %w", err))
	}
	return newConn, nil
}
func buildRepos(db *conn.DB) repository.Repositories {
	return repository.Repositories{
		User:         repository.NewUserRepo(db),
		RefreshToken: repository.NewRefreshTokenRepo(db),
		Card:         repository.NewCardRepo(db),
		Password:     repository.NewPasswordRepo(db),
		Note:         repository.NewNoteRepo(db),
		File:         repository.NewFileRepo(db),
	}

}

type serviceDeps struct {
	Repos   repository.Repositories
	Auth    *repository.AuthWriter
	Issuer  *token.Issuer
	Manager *trmanager.Manager
}

func buildServices(d serviceDeps) *service.Services {
	return &service.Services{
		User:     service.NewUserService(d.Repos.User, d.Auth, d.Issuer),
		Card:     service.NewCardService(d.Repos.Card),
		Password: service.NewPasswordService(d.Repos.Password),
		Note:     service.NewNoteService(d.Repos.Note),
		File:     service.NewFileService(d.Repos.File, d.Manager),
	}
}
