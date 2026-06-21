package passwordlist

import (
	"context"
	"encoding/json"
	"errors"
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

func enc(t *testing.T, v *vault.Vault, d clientmodel.PasswordData) []byte {
	t.Helper()
	raw, err := json.Marshal(d)
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
	rows    []repository.PasswordRow
	listErr error
}

func (f *fakeRepo) List(_ context.Context, _ string, _ int) ([]repository.PasswordRow, error) {
	return f.rows, f.listErr
}
func (f *fakeRepo) Update(context.Context, string, []byte) error { return nil }
func (f *fakeRepo) Delete(context.Context, string) error         { return nil }

func loadList(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init nil cmd")
	}

	for _, msg := range runCmd(t, cmd) {
		m, _ = m.Update(msg)
	}
	return m
}

func runCmd(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			if c == nil {
				continue
			}
			out = append(out, c())
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestListView(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{rows: []repository.PasswordRow{
		{ID: "p1", Data: enc(t, vlt, clientmodel.PasswordData{Login: "alice", Password: "secret", Meta: "work"}), Version: 1},
	}}
	m := New(Prop{Vault: vlt, Repo: repo})
	m = loadList(t, m)
	content := m.View().Content
	for _, want := range []string{"Passwords", "alice", "work", "LOGIN"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q: %q", want, content)
		}
	}
}

func TestListReveal(t *testing.T) {
	vlt := testVault(t)
	repo := &fakeRepo{rows: []repository.PasswordRow{
		{ID: "p1", Data: enc(t, vlt, clientmodel.PasswordData{Login: "alice", Password: "secret", Meta: "work"}), Version: 1},
	}}
	m := New(Prop{Vault: vlt, Repo: repo})
	m = loadList(t, m)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m.View().Content, "secret") {
		t.Fatalf("reveal view=%q", m.View().Content)
	}
}

func TestListEmpty(t *testing.T) {
	vlt := testVault(t)
	m := New(Prop{Vault: vlt, Repo: &fakeRepo{}})
	m = loadList(t, m)
	if !strings.Contains(m.View().Content, "No passwords") {
		t.Fatalf("empty view=%q", m.View().Content)
	}
}

func TestListLoadError(t *testing.T) {
	vlt := testVault(t)
	m := New(Prop{Vault: vlt, Repo: &fakeRepo{listErr: errors.New("dbfail")}})
	m = loadList(t, m)
	if !strings.Contains(m.View().Content, "Could not load") {
		t.Fatalf("err view=%q", m.View().Content)
	}
}

func TestDecodePasswordRoundTrip(t *testing.T) {
	vlt := testVault(t)
	want := clientmodel.PasswordData{Login: "u", Password: "p", Meta: "m"}
	row := repository.PasswordRow{ID: "id1", Data: enc(t, vlt, want), Version: 5}
	got, err := decodePassword(vlt, row)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "id1" || got.Version != 5 || got.Data != want {
		t.Fatalf("got %+v", got)
	}
}

func TestDecodePasswordCorrupt(t *testing.T) {
	vlt := testVault(t)
	if _, err := decodePassword(vlt, repository.PasswordRow{Data: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}

	bad, err := vlt.Encrypt([]byte("not json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodePassword(vlt, repository.PasswordRow{Data: bad}); err == nil {
		t.Fatal("expected json error")
	}
}

func TestMaskPassword(t *testing.T) {
	if maskPassword("") != "—" {
		t.Fatal("empty mask")
	}
	if maskPassword("anything") != "••••••••" {
		t.Fatal("nonempty mask")
	}
}

func TestRenderItem(t *testing.T) {
	got := renderItem(clientmodel.Password{Data: clientmodel.PasswordData{Login: "u", Password: "x", Meta: "y"}})
	if !strings.Contains(got, "u") || !strings.Contains(got, "y") {
		t.Fatalf("renderItem=%q", got)
	}

	got = renderItem(clientmodel.Password{})
	if !strings.Contains(got, "—") {
		t.Fatalf("empty renderItem=%q", got)
	}
}

func TestRenderDetail(t *testing.T) {
	got := renderDetail(clientmodel.Password{Data: clientmodel.PasswordData{Login: "u", Password: "p"}})
	if !strings.Contains(got, "Login") || !strings.Contains(got, "u") {
		t.Fatalf("detail=%q", got)
	}

	if !strings.Contains(got, "—") {
		t.Fatalf("detail missing dash for empty meta=%q", got)
	}
}
