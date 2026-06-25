package app

import (
	"context"
	"errors"
	"fmt"
	"gophkeeper/internal/server/auth/token"
	"gophkeeper/internal/server/config"
	"gophkeeper/internal/server/db/conn"
	"gophkeeper/internal/server/db/trmanager"
	"gophkeeper/internal/server/repository"
	"gophkeeper/internal/server/service"
	"gophkeeper/internal/server/transport"
	"gophkeeper/internal/shared/deps"
	"gophkeeper/internal/shared/errors/labelerrors"
	"gophkeeper/internal/shared/logger"
	mg "gophkeeper/migrations/server"
	"log/slog"
)

const (
	labelApp = "APP"
)

type dbCloser interface {
	Close() error
}

// appDeps holds the swappable leaf collaborators of Bootstrap. The ordered
// graph (repos → services → server) is wired from them; tests can override
// any of them via the With* options.
type appDeps struct {
	cfg *config.Flags
	log *slog.Logger
	db  *conn.DB
}

// Option configures Bootstrap. Each option validates at apply time and returns
// an error instead of failing later in the runtime.
type Option = deps.Option[appDeps]

// WithArgs loads the configuration from the command-line arguments.
func WithArgs(args []string) Option {
	return func(d *appDeps) error {
		cfg, err := config.Load(args)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		d.cfg = cfg
		return nil
	}
}

// WithLogger overrides the logger that Bootstrap would otherwise build.
func WithLogger(log *slog.Logger) Option {
	return func(d *appDeps) error {
		if log == nil {
			return errors.New("logger is nil")
		}
		d.log = log
		return nil
	}
}

// WithDB overrides the database connection that Bootstrap would otherwise open.
func WithDB(db *conn.DB) Option {
	return func(d *appDeps) error {
		if db == nil {
			return errors.New("db is nil")
		}
		d.db = db
		return nil
	}
}

type App struct {
	db     dbCloser
	server transport.Server
}

func Bootstrap(opts ...Option) (_ *App, err error) {
	d := &appDeps{}
	if err := deps.Apply(d, opts...); err != nil {
		return nil, err
	}

	if d.cfg == nil {
		return nil, errors.New("config is required (WithArgs)")
	}
	if d.log == nil {
		if d.log, err = logger.Initialize(&logger.Config{
			Mode:    d.cfg.AppMode,
			Dir:     d.cfg.LogDir,
			Prefix:  "server",
			Console: true,
		}); err != nil {
			return nil, fmt.Errorf("init logger: %w", err)
		}
	}
	if d.db == nil {
		if d.db, err = initDB(d.cfg); err != nil {
			return nil, fmt.Errorf("init db: %w", err)
		}
	}
	defer func() {
		if err != nil {
			_ = d.db.Close()
		}
	}()

	if err = mg.Migrate(d.cfg.DSN); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	mgr := trmanager.NewManager(d.db)
	repos := buildRepos(d.db)
	refreshIssuer := repository.NewRefreshIssuer(repos.RefreshToken, []byte(d.cfg.RefreshSecret), repository.DefaultRefreshTTL)
	authWriter := repository.NewAuthWriter(mgr, repos, refreshIssuer)
	issuer := token.NewIssuer([]byte(d.cfg.JWTSecret), token.DefaultAccessTTL)
	services := buildServices(serviceDeps{
		Repos:   repos,
		Auth:    authWriter,
		Issuer:  issuer,
		Manager: mgr,
	})

	server, err := transport.NewServer(
		transport.ServerProp{
			Config:      &transport.Config{Transport: d.cfg.Transport, Address: d.cfg.Address, CertFile: d.cfg.TLSCert, KeyFile: d.cfg.TLSKey},
			Services:    services,
			Logger:      d.log,
			TokenParser: issuer,
		})
	if err != nil {
		return nil, fmt.Errorf("new server: %w", err)
	}

	return &App{
		db:     d.db,
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
		Card:         repository.NewEntryRepo(db, repository.TableCard),
		Password:     repository.NewEntryRepo(db, repository.TablePassword),
		Note:         repository.NewEntryRepo(db, repository.TableNote),
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
		Card:     service.NewEntryService(d.Repos.Card),
		Password: service.NewEntryService(d.Repos.Password),
		Note:     service.NewEntryService(d.Repos.Note),
		File:     service.NewFileService(d.Repos.File, d.Manager),
	}
}
