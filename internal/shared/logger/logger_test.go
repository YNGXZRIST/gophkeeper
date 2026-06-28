package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitialize_Development(t *testing.T) {
	log, err := Initialize(&Config{Mode: ModeDevelopment, Dir: t.TempDir(), Prefix: "test"})
	if err != nil {
		t.Fatalf("Initialize development: %v", err)
	}
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	log.Info("hello")
}

func TestInitialize_Production(t *testing.T) {
	for _, console := range []bool{false, true} {
		dir := t.TempDir()
		log, err := Initialize(&Config{Mode: ModeProduction, Dir: dir, Prefix: "test", Console: console})
		if err != nil {
			t.Fatalf("Initialize production (console=%v): %v", console, err)
		}
		if log == nil {
			t.Fatalf("expected non-nil logger (console=%v)", console)
		}
		log.Info("info record")
		log.Error("error record")

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		if len(entries) == 0 {
			t.Fatalf("expected log files in %s (console=%v)", dir, console)
		}
	}
}

func TestInitialize_InvalidMode(t *testing.T) {
	log, err := Initialize(&Config{Mode: "bogus", Dir: t.TempDir(), Prefix: "test"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if log != nil {
		t.Fatal("expected nil logger on error")
	}
}

func TestInitialize_ProductionBadDir(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	badDir := filepath.Join(file, "sub")
	if _, err := Initialize(&Config{Mode: ModeProduction, Dir: badDir, Prefix: "test"}); err == nil {
		t.Fatal("expected error when log dir cannot be created")
	}
}
