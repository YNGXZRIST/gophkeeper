package migrator

import (
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/golang-migrate/migrate/v4/database"
	sqlite3driver "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

func runMigrations(t *testing.T, dbPath string, fsys fstest.MapFS) error {
	t.Helper()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var drv database.Driver
	drv, err = sqlite3driver.WithInstance(db, &sqlite3driver.Config{})
	if err != nil {
		t.Fatalf("sqlite driver: %v", err)
	}
	return Run(fsys, drv, "sqlite3")
}

func TestRun_AppliesMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fsys := fstest.MapFS{
		"0001_init.up.sql":   {Data: []byte("CREATE TABLE widgets (id INTEGER PRIMARY KEY);")},
		"0001_init.down.sql": {Data: []byte("DROP TABLE widgets;")},
	}
	if err := runMigrations(t, dbPath, fsys); err != nil {
		t.Fatalf("Run: %v", err)
	}

	verify, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open verify: %v", err)
	}
	defer func() {
		if cerr := verify.Close(); cerr != nil {
			t.Errorf("close verify: %v", cerr)
		}
	}()
	var name string
	row := verify.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='widgets'")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("expected widgets table created: %v", err)
	}
}

func TestRun_NoChange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fsys := fstest.MapFS{
		"0001_init.up.sql":   {Data: []byte("CREATE TABLE t1 (id INTEGER PRIMARY KEY);")},
		"0001_init.down.sql": {Data: []byte("DROP TABLE t1;")},
	}
	if err := runMigrations(t, dbPath, fsys); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	if err := runMigrations(t, dbPath, fsys); err != nil {
		t.Fatalf("second Run (no change): %v", err)
	}
}

func TestRun_BadMigrationSQL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fsys := fstest.MapFS{
		"0001_init.up.sql":   {Data: []byte("THIS IS NOT VALID SQL;")},
		"0001_init.down.sql": {Data: []byte("DROP TABLE nope;")},
	}
	if err := runMigrations(t, dbPath, fsys); err == nil {
		t.Fatal("expected error for invalid migration SQL")
	}
}

func TestRun_EmptyFSNoFiles(t *testing.T) {

	dbPath := filepath.Join(t.TempDir(), "test.db")
	if err := runMigrations(t, dbPath, fstest.MapFS{}); err == nil {
		t.Fatal("expected error for empty migration FS")
	}
}
