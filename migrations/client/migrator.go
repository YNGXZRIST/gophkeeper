package client

import (
	"embed"
	"fmt"
	conn "gophkeeper/internal/client/db"
	"gophkeeper/internal/shared/migrator"

	"github.com/golang-migrate/migrate/v4/database/sqlite"
)

//go:embed *sql

// FS holds migration SQL files embedded in the binary.
var FS embed.FS

// Migrate opens the SQLite database at PathDB and runs migrate.Up().
func Migrate() error {
	db, err := conn.Open()
	if err != nil {
		return err
	}
	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("sqlite driver: %w", err)
	}
	defer dbDriver.Close()

	return migrator.Run(FS, dbDriver, conn.Driver)
}
