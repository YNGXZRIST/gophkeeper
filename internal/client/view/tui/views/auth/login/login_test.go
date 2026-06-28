package login

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/crypto"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	"gophkeeper/internal/client/view/tui/views/auth/internal/credform"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeClient struct {
	userv1.UserServiceClient
	resp *userv1.LoginResponse
	err  error
}

func (c fakeClient) Login(_ context.Context, _ *userv1.LoginRequest, _ ...grpc.CallOption) (*userv1.LoginResponse, error) {
	return c.resp, c.err
}

type fakeStore struct {
	session *auth.Session
	err     error
	saved   bool
}

func (s *fakeStore) Save(_ context.Context, _ auth.Credentials) (*auth.Session, error) {
	s.saved = true
	return s.session, s.err
}

func goodResponse(t *testing.T, password string) (*userv1.LoginResponse, *auth.Session) {
	t.Helper()
	salt, err := crypto.GenerateBytes(16)
	if err != nil {
		t.Fatalf("salt: %v", err)
	}
	dek, err := crypto.GenerateBytes(32)
	if err != nil {
		t.Fatalf("dek: %v", err)
	}
	enc, err := crypto.NewEncryptor(crypto.DeriveKey(password, salt))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	wrapped, err := enc.Encrypt(dek)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	at, rt := "access", "refresh"
	login := "alice"
	resp := userv1.LoginResponse_builder{
		User:         userv1.User_builder{Login: &login}.Build(),
		AccessToken:  &at,
		RefreshToken: &rt,
		EncSalt:      salt,
		WrappedDek:   wrapped,
	}.Build()
	session := auth.NewSession("alice", auth.Token{}, auth.Token{}, salt, wrapped)
	return resp, session
}

func newModel(c userv1.UserServiceClient, store sessionStore, v *vault.Vault) tea.Model {
	return New(Prop{Client: c, Store: store, Vault: v})
}

func TestNewInitView(t *testing.T) {
	m := newModel(fakeClient{}, &fakeStore{}, vault.New())
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	if !strings.Contains(m.View().Content, "Login") {
		t.Fatal("View missing title")
	}
}

func TestUpdateCtrlC(t *testing.T) {
	m := newModel(fakeClient{}, &fakeStore{}, vault.New())
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %#v", cmd())
	}
}

func TestUpdateEscBack(t *testing.T) {
	m := newModel(fakeClient{}, &fakeStore{}, vault.New())
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, ok := cmd().(nav.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %#v", cmd())
	}
}

func TestUpdateFormPassthrough(t *testing.T) {
	m := newModel(fakeClient{}, &fakeStore{}, vault.New())
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	_ = cmd
	if strings.Contains(m.View().Content, "error") {
		t.Fatal("unexpected error before submit")
	}
}

func TestSubmitSuccess(t *testing.T) {
	resp, session := goodResponse(t, "pw")
	v := vault.New()
	store := &fakeStore{session: session}
	m := newModel(fakeClient{resp: resp}, store, v)

	_, cmd := m.Update(credform.SubmitMsg{Login: "alice", Password: "pw"})
	if cmd == nil {
		t.Fatal("expected reset cmd")
	}
	reset, ok := cmd().(nav.ResetMsg)
	if !ok || reset.ID != nav.Home {
		t.Fatalf("expected Reset(Home), got %#v", cmd())
	}
	if !store.saved {
		t.Fatal("store.Save was not called")
	}
	if v.Locked() {
		t.Fatal("vault should be unlocked")
	}
}

func TestSubmitUnauthenticated(t *testing.T) {
	m := newModel(fakeClient{err: status.Error(codes.Unauthenticated, "no")}, &fakeStore{}, vault.New())
	mm, cmd := m.Update(credform.SubmitMsg{Login: "a", Password: "b"})
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %#v", cmd())
	}
	if !strings.Contains(mm.View().Content, "Invalid login or password") {
		t.Fatalf("got %q", mm.View().Content)
	}
}

func TestSubmitUnavailable(t *testing.T) {
	m := newModel(fakeClient{err: status.Error(codes.Unavailable, "down")}, &fakeStore{}, vault.New())
	mm, _ := m.Update(credform.SubmitMsg{Login: "a", Password: "b"})
	if !strings.Contains(mm.View().Content, "Server error") {
		t.Fatalf("got %q", mm.View().Content)
	}
}

func TestSubmitInternal(t *testing.T) {
	m := newModel(fakeClient{err: status.Error(codes.Internal, "boom")}, &fakeStore{}, vault.New())
	mm, _ := m.Update(credform.SubmitMsg{Login: "a", Password: "b"})
	if !strings.Contains(mm.View().Content, "Internal error") {
		t.Fatalf("got %q", mm.View().Content)
	}
}

func TestSubmitStoreError(t *testing.T) {
	resp, _ := goodResponse(t, "pw")
	store := &fakeStore{err: errors.New("disk full")}
	m := newModel(fakeClient{resp: resp}, store, vault.New())
	mm, cmd := m.Update(credform.SubmitMsg{Login: "alice", Password: "pw"})
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %#v", cmd())
	}
	if !strings.Contains(mm.View().Content, "Client error") {
		t.Fatalf("got %q", mm.View().Content)
	}
}

func TestSubmitWrongPassword(t *testing.T) {

	resp, session := goodResponse(t, "pw")
	store := &fakeStore{session: session}
	m := newModel(fakeClient{resp: resp}, store, vault.New())
	mm, cmd := m.Update(credform.SubmitMsg{Login: "alice", Password: "other"})
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %#v", cmd())
	}
	if !strings.Contains(mm.View().Content, "Wrong password") {
		t.Fatalf("got %q", mm.View().Content)
	}
}
