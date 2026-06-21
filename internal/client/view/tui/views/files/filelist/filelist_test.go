package filelist

import (
	"context"
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	v := vault.New()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := v.UseDEK(key); err != nil {
		t.Fatalf("UseDEK: %v", err)
	}
	return v
}

func encodeMeta(t *testing.T, v *vault.Vault, meta clientmodel.FileMeta) []byte {
	t.Helper()
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	blob, err := v.Encrypt(raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return blob
}

type fakeRepo struct {
	rows []repository.FileRow
	err  error
}

func (r fakeRepo) List(_ context.Context, _ string, _ int) ([]repository.FileRow, error) {
	return r.rows, r.err
}
func (r fakeRepo) UpdateMeta(_ context.Context, _ string, _ []byte) error { return nil }
func (r fakeRepo) Delete(_ context.Context, _ string) error               { return nil }

func TestDecodeFileRoundTrip(t *testing.T) {
	v := testVault(t)
	meta := clientmodel.FileMeta{Name: "report.pdf", Meta: "work", Size: 2048}
	row := repository.FileRow{ID: "f-1", Meta: encodeMeta(t, v, meta), ChunkCount: 3, Version: 9}

	got, err := decodeFile(v, row)
	if err != nil {
		t.Fatalf("decodeFile: %v", err)
	}
	if got.ID != "f-1" || got.ChunkCount != 3 || got.Version != 9 {
		t.Errorf("unexpected meta fields: %+v", got)
	}
	if got.Meta != meta {
		t.Errorf("Meta = %+v, want %+v", got.Meta, meta)
	}
}

func TestDecodeFileCorrupt(t *testing.T) {
	v := testVault(t)
	row := repository.FileRow{ID: "f-2", Meta: []byte("garbage")}
	if _, err := decodeFile(v, row); err == nil {
		t.Fatal("expected error for corrupt ciphertext")
	}
}

func TestDecodeFileBadJSON(t *testing.T) {
	v := testVault(t)
	blob, err := v.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	row := repository.FileRow{ID: "f-3", Meta: blob}
	if _, err := decodeFile(v, row); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, c := range cases {
		if got := humanSize(c.in); got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderItem(t *testing.T) {
	full := renderItem(clientmodel.File{Meta: clientmodel.FileMeta{Name: "a.txt", Meta: "note", Size: 100}})
	if !strings.Contains(full, "a.txt") || !strings.Contains(full, "note") {
		t.Errorf("renderItem missing fields: %q", full)
	}
	empty := renderItem(clientmodel.File{})
	if !strings.Contains(empty, "—") {
		t.Errorf("renderItem empty should use placeholder: %q", empty)
	}
}

func TestRenderDetail(t *testing.T) {
	full := renderDetail(clientmodel.File{Meta: clientmodel.FileMeta{Name: "a.txt", Meta: "note", Size: 100}})
	if !strings.Contains(full, "Name") || !strings.Contains(full, "a.txt") {
		t.Errorf("renderDetail missing fields: %q", full)
	}
	empty := renderDetail(clientmodel.File{})
	if !strings.Contains(empty, "—") {
		t.Errorf("renderDetail empty should use placeholder: %q", empty)
	}
}

func TestNewWiring(t *testing.T) {
	v := testVault(t)
	meta := clientmodel.FileMeta{Name: "x.bin", Size: 10}
	repo := fakeRepo{rows: []repository.FileRow{{ID: "f-1", Meta: encodeMeta(t, v, meta), ChunkCount: 1, Version: 1}}}

	m := New(Prop{Vault: v, Repo: repo})
	if m == nil {
		t.Fatal("New returned nil model")
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	msg := cmd()
	m2, _ := m.Update(msg)
	view := m2.View()
	if view.Content == "" {
		t.Error("View content empty after fetch")
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_ = m3.View()
}
