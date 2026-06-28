package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gophkeeper/internal/server/config"
	"gophkeeper/internal/server/db/conn"
)

type fakeServer struct {
	startErr error
	stopErr  error
	started  bool
	stopped  bool
}

func (f *fakeServer) Start() error {
	f.started = true
	return f.startErr
}

func (f *fakeServer) Stop(ctx context.Context) error {
	f.stopped = true
	return f.stopErr
}

type fakeCloser struct {
	err    error
	closed bool
}

func (f *fakeCloser) Close() error {
	f.closed = true
	return f.err
}

func TestInitDB(t *testing.T) {
	t.Run("empty DSN errors", func(t *testing.T) {
		if _, err := initDB(&config.Flags{DSN: ""}); err == nil {
			t.Fatal("expected error for empty DSN")
		}
	})

	t.Run("valid DSN constructs lazily", func(t *testing.T) {
		db, err := initDB(&config.Flags{DSN: "postgres://user:pass@localhost:5432/db?sslmode=disable"})
		if err != nil {
			t.Fatalf("initDB: %v", err)
		}
		if db == nil {
			t.Fatal("nil db")
		}
		t.Cleanup(func() { _ = db.Close() })
	})
}

func newTestConn(t *testing.T) *conn.DB {
	t.Helper()
	db, err := conn.NewConn(conn.NewCfg("postgres://u:p@localhost:5432/d?sslmode=disable"))
	if err != nil {
		t.Fatalf("NewConn: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestBuildRepos(t *testing.T) {
	repos := buildRepos(newTestConn(t))
	if repos.User == nil || repos.RefreshToken == nil || repos.Card == nil ||
		repos.Password == nil || repos.Note == nil || repos.File == nil {
		t.Fatalf("buildRepos left a nil repo: %+v", repos)
	}
}

func TestBuildServices(t *testing.T) {
	db := newTestConn(t)
	repos := buildRepos(db)
	svc := buildServices(serviceDeps{Repos: repos})
	if svc == nil {
		t.Fatal("nil services")
	}
	if svc.User == nil || svc.Card == nil || svc.Password == nil || svc.Note == nil || svc.File == nil {
		t.Fatalf("buildServices left a nil service: %+v", svc)
	}
}

func TestAppRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fs := &fakeServer{}
		a := &App{db: &fakeCloser{}, server: fs}
		if err := a.Run(); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if !fs.started {
			t.Fatal("server not started")
		}
	})

	t.Run("start error propagated", func(t *testing.T) {
		want := errors.New("boom")
		a := &App{db: &fakeCloser{}, server: &fakeServer{startErr: want}}
		if err := a.Run(); !errors.Is(err, want) {
			t.Fatalf("Run err = %v, want %v", err, want)
		}
	})
}

func TestAppShutdown(t *testing.T) {
	ctx := context.Background()

	t.Run("clean shutdown", func(t *testing.T) {
		fs := &fakeServer{}
		fc := &fakeCloser{}
		a := &App{db: fc, server: fs}
		if err := a.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
		if !fs.stopped || !fc.closed {
			t.Fatal("stop/close not called")
		}
	})

	t.Run("stop error returned", func(t *testing.T) {
		stopErr := errors.New("stop fail")
		a := &App{db: &fakeCloser{}, server: &fakeServer{stopErr: stopErr}}
		if err := a.Shutdown(ctx); !errors.Is(err, stopErr) {
			t.Fatalf("Shutdown err = %v, want %v", err, stopErr)
		}
	})

	t.Run("close error returned when stop ok", func(t *testing.T) {
		closeErr := errors.New("close fail")
		a := &App{db: &fakeCloser{err: closeErr}, server: &fakeServer{}}
		if err := a.Shutdown(ctx); !errors.Is(err, closeErr) {
			t.Fatalf("Shutdown err = %v, want %v", err, closeErr)
		}
	})

	t.Run("stop error wins over close error", func(t *testing.T) {
		stopErr := errors.New("stop fail")
		a := &App{db: &fakeCloser{err: errors.New("close fail")}, server: &fakeServer{stopErr: stopErr}}
		if err := a.Shutdown(ctx); !errors.Is(err, stopErr) {
			t.Fatalf("Shutdown err = %v, want %v", err, stopErr)
		}
	})
}

func TestBootstrapOptionValidation(t *testing.T) {
	t.Run("WithDB nil", func(t *testing.T) {
		_, err := Bootstrap(WithDB(nil))
		if err == nil || !strings.Contains(err.Error(), "db is nil") {
			t.Fatalf("want db-nil error, got %v", err)
		}
	})

	t.Run("WithLogger nil", func(t *testing.T) {
		_, err := Bootstrap(WithLogger(nil))
		if err == nil || !strings.Contains(err.Error(), "logger is nil") {
			t.Fatalf("want logger-nil error, got %v", err)
		}
	})
}

func TestBootstrapConfigError(t *testing.T) {

	if _, err := Bootstrap(WithArgs([]string{"--this-flag-does-not-exist"})); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestBootstrapMigrateError(t *testing.T) {

	t.Setenv("JWT_SECRET", "jwt-test-secret")
	t.Setenv("REFRESH_SECRET", "refresh-test-secret")
	args := []string{
		"-t", "grpc",
		"-a", "127.0.0.1:0",
		"-m", "development",
		"-l", t.TempDir(),
		"-d", "postgres://u:p@127.0.0.1:1/nodb?sslmode=disable&connect_timeout=1",
	}
	app, err := Bootstrap(WithArgs(args))
	if err == nil {
		if app != nil {
			_ = app.Shutdown(context.Background())
		}
		t.Fatal("expected migrate error against unreachable database")
	}
}
