package cardadd

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
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
	created []byte
	err     error
}

func (f *fakeRepo) Create(_ context.Context, data []byte) (repository.CardRow, error) {
	f.created = data
	return repository.CardRow{ID: "new"}, f.err
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

func filledCard() clientmodel.CardData {
	return clientmodel.CardData{Number: "4111111111111111", Holder: "JD", Expiry: "12/30", CVV: "123", Meta: "m"}
}

func TestNewCreatesModel(t *testing.T) {
	repo := &fakeRepo{}
	m := New(Prop{Vault: testVault(t), Repo: repo})
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.Init() == nil {
		t.Fatal("Init nil")
	}
}

func TestSubmitCallsCreateAndNavSequence(t *testing.T) {
	repo := &fakeRepo{}
	v := testVault(t)

	m := New(Prop{Vault: v, Repo: repo})

	for _, r := range filledCard().Number {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(specialKey(tea.KeyDown))
	for _, r := range "JD" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(specialKey(tea.KeyDown))
	for _, r := range "12/30" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(specialKey(tea.KeyDown))
	for _, r := range "123" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(specialKey(tea.KeyDown))
	for _, r := range "meta" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	m, _ = m.Update(specialKey(tea.KeyDown))
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("submit produced nil cmd")
	}

	msg := cmd()

	m2, navCmd := m.Update(msg)
	_ = m2
	if navCmd == nil {
		t.Fatal("expected nav sequence after save")
	}
	if len(repo.created) == 0 {
		t.Fatal("Create was not called")
	}
	msgs := collectMsgs(navCmd)
	if len(msgs) < 3 {
		t.Fatalf("expected back/reload/syncnow, got %#v", msgs)
	}
}

func TestSubmitCreateError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("boom")}
	m := New(Prop{Vault: testVault(t), Repo: repo})
	for _, r := range filledCard().Number {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	for _, fld := range []string{"JD", "12/30", "123", "meta"} {
		m, _ = m.Update(specialKey(tea.KeyDown))
		for _, r := range fld {
			m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
	}
	m, _ = m.Update(specialKey(tea.KeyDown))
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	msg := cmd()
	if len(repo.created) == 0 {
		t.Fatal("Create not called")
	}

	_, navCmd := m.Update(msg)
	if navCmd != nil {
		t.Fatal("expected nil cmd on create error")
	}
}

func TestEscBack(t *testing.T) {
	m := New(Prop{Vault: testVault(t), Repo: &fakeRepo{}})
	_, cmd := m.Update(specialKey(tea.KeyEscape))
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}
