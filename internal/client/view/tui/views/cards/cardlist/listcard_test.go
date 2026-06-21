package cardlist

import (
	"context"
	"encoding/json"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeRepo struct {
	rows      []repository.CardRow
	listErr   error
	deletedID string
	deleteErr error
}

func (f *fakeRepo) List(_ context.Context, _ string, _ int) ([]repository.CardRow, error) {
	return f.rows, f.listErr
}
func (f *fakeRepo) Update(context.Context, string, []byte) error { return nil }
func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.deletedID = id
	return f.deleteErr
}

func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }
func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func encRow(t *testing.T, v *vault.Vault, id string, d clientmodel.CardData) repository.CardRow {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	blob, err := v.Encrypt(raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return repository.CardRow{ID: id, Data: blob, Version: 1}
}

func loadList(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned nil")
	}

	for _, msg := range runBatch(cmd) {
		next, _ := m.Update(msg)
		m = next
	}
	return m
}

func runBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, runOne(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func runOne(c tea.Cmd) []tea.Msg {
	if c == nil {
		return nil
	}
	return []tea.Msg{c()}
}

func TestNewAndFetchWiring(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.CardRow{
		encRow(t, v, "c1", clientmodel.CardData{Number: "4111111111111111", Holder: "ALICE", Expiry: "12/30", Meta: "work"}),
		encRow(t, v, "c2", clientmodel.CardData{Number: "4222222222222222", Holder: "BOB"}),
	}}
	m := loadList(t, New(Prop{Vault: v, Repo: repo}))
	content := m.View().Content
	if !strings.Contains(content, "ALICE") {
		t.Fatalf("view missing decoded item: %q", content)
	}
	if !strings.Contains(content, "1111") {
		t.Fatalf("view missing masked number: %q", content)
	}
}

func TestFetchError(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{listErr: errors.New("db down")}
	m := loadList(t, New(Prop{Vault: v, Repo: repo}))
	if !strings.Contains(m.View().Content, "Could not load") {
		t.Fatalf("expected load error view, got %q", m.View().Content)
	}
}

func TestRevealDetail(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.CardRow{
		encRow(t, v, "c1", clientmodel.CardData{Number: "4111111111111111", Holder: "ALICE", Expiry: "12/30", CVV: "123", Meta: "work"}),
	}}
	m := loadList(t, New(Prop{Vault: v, Repo: repo}))
	m, _ = m.Update(specialKey(tea.KeyEnter))
	if !strings.Contains(m.View().Content, "Number") {
		t.Fatalf("detail not rendered: %q", m.View().Content)
	}
}

func TestEditWiring(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.CardRow{
		encRow(t, v, "c1", clientmodel.CardData{Number: "4111111111111111", Holder: "ALICE"}),
	}}
	m := loadList(t, New(Prop{Vault: v, Repo: repo}))
	_, cmd := m.Update(runeKey('e'))
	if cmd == nil {
		t.Fatal("e should push edit model")
	}
	msg := cmd()
	pm, ok := msg.(nav.PushModelMsg)
	if !ok {
		t.Fatalf("expected PushModelMsg, got %T", msg)
	}
	if pm.Model == nil {
		t.Fatal("NewEdit produced nil model")
	}
}

func TestDeleteWiring(t *testing.T) {
	v := testVault(t)
	repo := &fakeRepo{rows: []repository.CardRow{
		encRow(t, v, "c1", clientmodel.CardData{Number: "4111111111111111", Holder: "ALICE"}),
	}}
	m := loadList(t, New(Prop{Vault: v, Repo: repo}))
	m, _ = m.Update(runeKey('d'))
	if !strings.Contains(m.View().Content, "delete selected") {
		t.Fatalf("confirm prompt missing: %q", m.View().Content)
	}
	_, cmd := m.Update(runeKey('y'))
	if cmd == nil {
		t.Fatal("y should trigger remove")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("remove produced nil msg")
	}
	if repo.deletedID != "c1" {
		t.Fatalf("Delete id = %q, want c1", repo.deletedID)
	}
}

func TestAddAndConflictNav(t *testing.T) {
	v := testVault(t)
	m := loadList(t, New(Prop{Vault: v, Repo: &fakeRepo{}}))
	_, addCmd := m.Update(runeKey('a'))
	if pm, ok := addCmd().(nav.PushMsg); !ok || pm.ID != nav.CardAdd {
		t.Fatalf("expected push CardAdd, got %#v", addCmd())
	}
	_, cCmd := m.Update(runeKey('c'))
	if pm, ok := cCmd().(nav.PushMsg); !ok || pm.ID != nav.CardSync {
		t.Fatalf("expected push CardSync, got %#v", cCmd())
	}
}

func TestRenderItemEmptyFields(t *testing.T) {
	got := renderItem(clientmodel.Card{Data: clientmodel.CardData{Number: "4111111111111111"}})
	if !strings.Contains(got, "—") {
		t.Fatalf("empty holder/meta should render dash: %q", got)
	}
}

func TestRenderDetailEmptyFields(t *testing.T) {
	got := renderDetail(clientmodel.Card{Data: clientmodel.CardData{Number: "4111111111111111"}})
	if !strings.Contains(got, "—") {
		t.Fatalf("empty fields should render dash: %q", got)
	}
}
