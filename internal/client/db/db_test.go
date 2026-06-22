package db

import (
	"os"
	"testing"
)

func TestOpen(t *testing.T) {

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if cerr := os.Chdir(wd); cerr != nil {
			t.Errorf("restore cwd: %v", cerr)
		}
	})

	conn, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			t.Errorf("close: %v", cerr)
		}
	}()

	if err := conn.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestConstants(t *testing.T) {
	if Driver != "sqlite" {
		t.Fatalf("Driver = %q", Driver)
	}
	if PathDB == "" {
		t.Fatal("PathDB empty")
	}
}
