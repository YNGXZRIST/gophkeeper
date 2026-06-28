package fileedit

import (
	"context"
	"errors"
	clientmodel "gophkeeper/internal/client/model"
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

type fakeRepo struct {
	err     error
	gotID   string
	gotMeta []byte
}

func (r *fakeRepo) UpdateMeta(_ context.Context, id string, meta []byte) error {
	r.gotID = id
	r.gotMeta = meta
	return r.err
}

func newModel(t *testing.T, repo *fakeRepo) model {
	t.Helper()
	m := New(Prop{
		Vault: testVault(t),
		Repo:  repo,
		File:  clientmodel.File{ID: "f-1", Meta: clientmodel.FileMeta{Name: "a.txt", Size: 5, Meta: "old"}},
	})
	return m.(model)
}

func TestInitView(t *testing.T) {
	m := newModel(t, &fakeRepo{})
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil")
	}
	if v := m.View(); v.Content == "" {
		t.Error("View empty")
	}
}

func TestEscBack(t *testing.T) {
	m := newModel(t, &fakeRepo{})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should produce Back cmd")
	}
}

func TestCtrlCQuit(t *testing.T) {
	m := newModel(t, &fakeRepo{})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c should produce Quit cmd")
	}
}

func TestSubmitSuccess(t *testing.T) {
	repo := &fakeRepo{}
	m := newModel(t, repo)
	cmd := m.submit()
	msg := cmd()
	saved, ok := msg.(savedMsg)
	if !ok {
		t.Fatalf("got %T, want savedMsg", msg)
	}
	if saved.err != nil {
		t.Fatalf("unexpected err: %v", saved.err)
	}
	if repo.gotID != "f-1" {
		t.Errorf("UpdateMeta id = %q, want f-1", repo.gotID)
	}
	if len(repo.gotMeta) == 0 {
		t.Error("expected encrypted meta passed to repo")
	}

	_, seq := m.Update(savedMsg{})
	if seq == nil {
		t.Fatal("expected sequence cmd on successful save")
	}
}

func TestSubmitRepoError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("boom")}
	m := newModel(t, repo)
	msg := m.submit()()
	saved, ok := msg.(savedMsg)
	if !ok || saved.err == nil {
		t.Fatalf("expected savedMsg with error, got %#v", msg)
	}

	m2, cmd := m.Update(saved)
	if cmd != nil {
		t.Error("error path should not navigate")
	}
	if !strings.Contains(m2.View().Content, "Save failed") {
		t.Error("expected error message in view")
	}
}

func TestSubmitViaFormAction(t *testing.T) {
	repo := &fakeRepo{}
	m := newModel(t, repo)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2, cmd := next.(model).Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on the submit button should yield a submit cmd")
	}
	if _, ok := m2.(model); !ok {
		t.Fatalf("unexpected model type %T", m2)
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(savedMsg); !ok {
			t.Fatalf("submit cmd should yield savedMsg, got %T", msg)
		}
	}
}
