package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

var dbCounter atomic.Int64

func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		candidate := filepath.Join(dir, "migrations", "client")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("migrations/client not found from working directory")
		}
		dir = parent
	}
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	require.NotEmpty(t, ups)

	sortStrings(ups)

	for _, name := range ups {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		if _, err := db.Exec(string(raw)); err != nil {

			for _, stmt := range strings.Split(string(raw), ";") {
				s := strings.TrimSpace(stmt)
				if s == "" {
					continue
				}
				_, execErr := db.Exec(s)
				require.NoErrorf(t, execErr, "migration %s statement failed", name)
			}
		}
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + name + "_" + itoa(dbCounter.Add(1)) + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	applyMigrations(t, db)
	return db
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

type rowState struct {
	Version       int64
	BaseVersion   int64
	Dirty         int
	Deleted       int
	Conflict      int
	ServerBlob    []byte
	ServerVersion sql.NullInt64
}

func readState(t *testing.T, db *sql.DB, table, id string) rowState {
	t.Helper()
	var st rowState
	q := "SELECT version, base_version, dirty, deleted, conflict, server_blob, server_version FROM " + table + " WHERE id = ?"
	err := db.QueryRowContext(context.Background(), q, id).
		Scan(&st.Version, &st.BaseVersion, &st.Dirty, &st.Deleted, &st.Conflict, &st.ServerBlob, &st.ServerVersion)
	require.NoError(t, err)
	return st
}

func rowExists(t *testing.T, db *sql.DB, table, id string) bool {
	t.Helper()
	var n int
	err := db.QueryRowContext(context.Background(), "SELECT COUNT(1) FROM "+table+" WHERE id = ?", id).Scan(&n)
	require.NoError(t, err)
	return n > 0
}
