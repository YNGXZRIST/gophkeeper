package syncnotes

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/sync/syncer"

	_ "modernc.org/sqlite"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	dir := "../../../../migrations/client"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	var ups []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".sql" && len(name) > 7 && name[len(name)-7:] == ".up.sql" {
			ups = append(ups, name)
		}
	}
	sort.Strings(ups)
	for _, name := range ups {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Fatalf("exec %s: %v", name, err)
		}
	}
	return db
}

func TestRepoDelegates(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	notes := repository.NewNotesRepo(db)
	r := NewRepo(notes, repository.NewSyncStateRepo(db))

	var _ syncer.Repo = r

	if err := r.Upsert(ctx, "n1", []byte("hello"), 3); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	row, ok, err := r.Get(ctx, "n1")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if string(row.Data) != "hello" || row.Version != 3 || row.BaseVersion != 3 || row.Dirty {
		t.Fatalf("row = %+v", row)
	}

	_, ok, err = r.Get(ctx, "missing")
	if err != nil || ok {
		t.Fatalf("get missing: ok=%v err=%v", ok, err)
	}

	created, err := notes.Create(ctx, []byte("local"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	dirty, err := r.ListDirty(ctx)
	if err != nil {
		t.Fatalf("list dirty: %v", err)
	}
	if len(dirty) != 1 || dirty[0].ID != created.ID || !dirty[0].Dirty {
		t.Fatalf("dirty = %+v", dirty)
	}

	if err = r.MarkSynced(ctx, created.ID, 7); err != nil {
		t.Fatalf("mark synced: %v", err)
	}
	row, _, err = r.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get after sync: %v", err)
	}
	if row.Dirty || row.Version != 7 || row.BaseVersion != 7 {
		t.Fatalf("after synced = %+v", row)
	}

	if err = r.MarkConflict(ctx, "n1", []byte("srv"), 9); err != nil {
		t.Fatalf("mark conflict: %v", err)
	}

	if err = r.HardDelete(ctx, "n1"); err != nil {
		t.Fatalf("hard delete: %v", err)
	}
	if _, ok, _ := r.Get(ctx, "n1"); ok {
		t.Fatal("row should be deleted")
	}

	cur, err := r.LastSyncedAt(ctx)
	if err != nil || cur != "" {
		t.Fatalf("initial cursor = %q err=%v", cur, err)
	}
	if err = r.SetLastSyncedAt(ctx, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("set cursor: %v", err)
	}
	cur, err = r.LastSyncedAt(ctx)
	if err != nil || cur != "2026-01-01T00:00:00Z" {
		t.Fatalf("cursor = %q err=%v", cur, err)
	}
}
