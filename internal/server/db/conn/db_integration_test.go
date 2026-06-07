//go:build integration
// +build integration

package conn

import (
	"database/sql"
	"errors"
	"gophermart-loyalty/internal/gopherman/config/db"
	"gophermart-loyalty/migrations"
	"os"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var testDB *DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		panic(err)
	}
	_ = resource.Expire(30)

	hostPort := resource.GetPort("5432/tcp")
	dsn := "postgres://postgres:postgres@localhost:" + hostPort + "/postgres?sslmode=disable"

	if err = pool.Retry(func() error {
		conn, retryErr := sql.Open("pgx", dsn)
		if retryErr != nil {
			return retryErr
		}
		defer conn.Close()
		return conn.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		panic(err)
	}
	if err = migrations.Migrate(dsn); err != nil {
		_ = pool.Purge(resource)
		panic(err)
	}

	var connErr error
	testDB, connErr = NewConn(db.NewCfg(dsn))
	if connErr != nil {
		_ = pool.Purge(resource)
		panic(connErr)
	}

	code := m.Run()
	_ = testDB.Close()
	_ = pool.Purge(resource)
	os.Exit(code)
}

func getConnForMethodTests(t *testing.T) *DB {
	t.Helper()
	if testDB == nil {
		t.Skip("integration DB not available (run with -tags=integration)")
	}
	return testDB
}

func TestDB_ExecQuery_methods(t *testing.T) {
	conn := getConnForMethodTests(t)
	ctx := t.Context()

	tests := []struct {
		name string
		run  func(*DB) error
	}{
		{
			name: "Exec",
			run: func(d *DB) error {
				_, err := d.Exec("SELECT 1")
				return err
			},
		},
		{
			name: "ExecContext",
			run: func(d *DB) error {
				_, err := d.ExecContext(ctx, "SELECT 1")
				return err
			},
		},
		{
			name: "Query",
			run: func(d *DB) error {
				rows, err := d.Query("SELECT 1")
				if err != nil {
					return err
				}
				defer rows.Close()
				if !rows.Next() {
					return errNoRow
				}
				var n int
				if err := rows.Scan(&n); err != nil {
					return err
				}
				if n != 1 {
					t.Errorf("Query: got row value %d, want 1", n)
				}
				if rows.Next() {
					t.Error("Query: expected single row")
				}
				return nil
			},
		},
		{
			name: "QueryContext",
			run: func(d *DB) error {
				rows, err := d.QueryContext(ctx, "SELECT 1")
				if err != nil {
					return err
				}
				defer rows.Close()
				if !rows.Next() {
					return errNoRow
				}
				var n int
				if err := rows.Scan(&n); err != nil {
					return err
				}
				if n != 1 {
					t.Errorf("QueryContext: got row value %d, want 1", n)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(conn); err != nil {
				t.Errorf("%s(SELECT 1) = %v", tt.name, err)
			}
		})
	}
}

var errNoRow = errors.New("expected one row")
