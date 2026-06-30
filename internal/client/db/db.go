package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const (
	// Driver is the database/sql driver name of the pure-Go SQLite implementation.
	Driver = "sqlite"

	// PathDB is the on-disk location of the client SQLite database.
	PathDB = "./app.db"
)

// Open opens the client SQLite database at PathDB.
func Open() (*sql.DB, error) {
	db, err := sql.Open(Driver, PathDB)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}
