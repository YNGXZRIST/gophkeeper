package cardedit

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"reflect"
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

type fakeRepo struct {
	gotID   string
	gotData []byte
	err     error
}

func (f *fakeRepo) Update(_ context.Context, id string, data []byte) error {
	f.gotID = id
	f.gotData = data
	return f.err
}

func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	return flatten(cmd())
}

func flatten(msg tea.Msg) []tea.Msg {
	if b, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range b {
			out = append(out, collectMsgs(c)...)
		}
		return out
	}
	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Type().Elem() == reflect.TypeOf(tea.Cmd(nil)) {
		var out []tea.Msg
		for i := 0; i < rv.Len(); i++ {
			out = append(out, collectMsgs(rv.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func seededCard() clientmodel.Card {
	return clientmodel.Card{
		ID: "card-42",
		Data: clientmodel.CardData{
			Number: "4111111111111111", Holder: "JD", Expiry: "12/30", CVV: "123", Meta: "m",
		},
	}
}

func TestNew(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Card: seededCard()})
	if m == nil || m.Init() == nil {
		t.Fatal("New/Init failed")
	}
}

func submit(t *testing.T, m tea.Model) (tea.Model, tea.Cmd) {
	t.Helper()
	for i := 0; i < 5; i++ {
		m, _ = m.Update(specialKey(tea.KeyDown))
	}
	return m.Update(specialKey(tea.KeyEnter))
}

func TestSubmitCallsUpdate(t *testing.T) {
	repo := &fakeRepo{}
	m := New(Prop{Vault: testVault(t), Repo: repo, Card: seededCard()})
	m2, cmd := submit(t, m)
	if cmd == nil {
		t.Fatal("submit nil cmd")
	}
	msg := cmd()
	if repo.gotID != "card-42" {
		t.Fatalf("Update id = %q, want card-42", repo.gotID)
	}
	if len(repo.gotData) == 0 {
		t.Fatal("Update data empty")
	}
	_, navCmd := m2.Update(msg)
	if navCmd == nil {
		t.Fatal("expected nav sequence after save")
	}
	if len(collectMsgs(navCmd)) < 3 {
		t.Fatal("expected back/reload/syncnow")
	}
}

func TestSubmitUpdateError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("nope")}
	m := New(Prop{Vault: testVault(t), Repo: repo, Card: seededCard()})
	m2, cmd := submit(t, m)
	msg := cmd()
	if repo.gotID == "" {
		t.Fatal("Update not called")
	}
	_, navCmd := m2.Update(msg)
	if navCmd != nil {
		t.Fatal("expected nil cmd on error")
	}
}

func TestEscBack(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}, Card: seededCard()})
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}
