package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const (
	Sqlite3 = "sqlite3"

	PathDB = "./app.db"
)

func Open() (*sql.DB, error) {
	db, err := sql.Open(Sqlite3, PathDB)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return db, nil
}
