package server

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *sql

// FS holds migration SQL files embedded in the binary.
var FS embed.FS

const (
	PGX      = "pgx"
	Postgres = "postgres"
)

// Migrate opens a connection with dsn and runs migrate.Up(); an empty dsn is an error.
func Migrate(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("database DSN is not set")
	}

	db, err := sql.Open(PGX, dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}
	defer dbDriver.Close()
	srcDriver, err := iofs.New(FS, ".")
	if err != nil {
		return fmt.Errorf("ifs source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", srcDriver, Postgres, dbDriver)
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
