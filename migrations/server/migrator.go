package server

import (
	"database/sql"
	"embed"
	"fmt"

	"gophkeeper/internal/shared/migrator"

	"github.com/golang-migrate/migrate/v4/database/postgres"
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

	return migrator.Run(FS, dbDriver, Postgres)
}
