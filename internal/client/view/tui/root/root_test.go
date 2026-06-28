package root

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gophkeeper/internal/client/auth"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"gophkeeper/internal/client/view/tui/components/nav"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "modernc.org/sqlite"
)

func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	v := vault.New()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := v.UseDEK(key); err != nil {
		t.Fatal(err)
	}
	return v
}

var dbCounter atomic.Int64

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:roottest_" + name + "_" + itoa(dbCounter.Add(1)) + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	applyMigrations(t, db)
	return db
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var ups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	if len(ups) == 0 {
		t.Fatal("no up migrations found")
	}
	for i := 1; i < len(ups); i++ {
		for j := i; j > 0 && ups[j-1] > ups[j]; j-- {
			ups[j-1], ups[j] = ups[j], ups[j-1]
		}
	}
	for _, name := range ups {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(raw)); err != nil {
			for _, stmt := range strings.Split(string(raw), ";") {
				s := strings.TrimSpace(stmt)
				if s == "" {
					continue
				}
				if _, execErr := db.Exec(s); execErr != nil {
					t.Fatalf("migration %s statement failed: %v", name, execErr)
				}
			}
		}
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "migrations", "client")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("migrations/client not found")
		}
		dir = parent
	}
}

type fakeStore struct {
	session   *auth.Session
	getErr    error
	saveErr   error
	clearErr  error
	saved     *auth.Credentials
	cleared   bool
	saveCalls int
}

func (f *fakeStore) Get(ctx context.Context) (*auth.Session, error) {
	return f.session, f.getErr
}

func (f *fakeStore) Save(ctx context.Context, cred auth.Credentials) (*auth.Session, error) {
	f.saveCalls++
	c := cred
	f.saved = &c
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	f.session = auth.NewSession(cred.Login, auth.Token{Raw: cred.AccessToken}, auth.Token{Raw: cred.RefreshToken}, cred.EncSalt, cred.WrappedDek)
	return f.session, nil
}

func (f *fakeStore) Clear(ctx context.Context) error {
	f.cleared = true
	return f.clearErr
}

type fakeSyncer struct {
	err    error
	called int32
}

func (f *fakeSyncer) SyncAll(ctx context.Context) error {
	atomic.AddInt32(&f.called, 1)
	return f.err
}

type fakeLogger struct{ errors int }

func (f *fakeLogger) Error(msg string, args ...any) { f.errors++ }

type fakeUserClient struct {
	userv1.UserServiceClient
	resp *userv1.RefreshResponse
	err  error
}

func (f *fakeUserClient) Refresh(ctx context.Context, in *userv1.RefreshRequest, opts ...grpc.CallOption) (*userv1.RefreshResponse, error) {
	return f.resp, f.err
}

func futureToken(t *testing.T) auth.Token {
	t.Helper()
	return auth.Token{Raw: "access", ExpiresAt: time.Now().Add(time.Hour)}
}

func expiredToken(t *testing.T) auth.Token {
	t.Helper()
	return auth.Token{Raw: "access", ExpiresAt: time.Now().Add(-time.Hour)}
}

func baseDeps(t *testing.T) Deps {
	t.Helper()
	db := newTestDB(t)
	return Deps{
		Vault:         testVault(t),
		NotesRepo:     repository.NewEntryRepo(db, repository.TableNote),
		CardsRepo:     repository.NewEntryRepo(db, repository.TableCard),
		PasswordsRepo: repository.NewEntryRepo(db, repository.TablePassword),
		FilesRepo:     repository.NewFilesRepo(db),
		Sync:          &fakeSyncer{},
		Logger:        &fakeLogger{},
		SessionStore:  &fakeStore{},
	}
}

func TestResolveStart(t *testing.T) {
	t.Run("no session -> Welcome", func(t *testing.T) {
		d := baseDeps(t)
		d.SessionStore = &fakeStore{session: nil}
		if got := resolveStart(d); got != nav.Welcome {
			t.Fatalf("got %v, want Welcome", got)
		}
	})

	t.Run("get error -> Welcome", func(t *testing.T) {
		d := baseDeps(t)
		d.SessionStore = &fakeStore{getErr: errors.New("boom")}
		if got := resolveStart(d); got != nav.Welcome {
			t.Fatalf("got %v, want Welcome", got)
		}
	})

	t.Run("valid unlocked -> Home", func(t *testing.T) {
		d := baseDeps(t)
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
		if got := resolveStart(d); got != nav.Home {
			t.Fatalf("got %v, want Home", got)
		}
	})

	t.Run("valid but locked -> Unlock", func(t *testing.T) {
		d := baseDeps(t)
		d.Vault = vault.New()
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
		if got := resolveStart(d); got != nav.Unlock {
			t.Fatalf("got %v, want Unlock", got)
		}
	})

	t.Run("expired no refresh -> Welcome", func(t *testing.T) {
		d := baseDeps(t)
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: expiredToken(t), Refresh: auth.Token{Raw: ""}}}
		if got := resolveStart(d); got != nav.Welcome {
			t.Fatalf("got %v, want Welcome", got)
		}
	})

	t.Run("expired refresh ok unlocked -> Home", func(t *testing.T) {
		d := baseDeps(t)
		resp := &userv1.RefreshResponse{}
		resp.SetAccessToken("newaccess")
		resp.SetRefreshToken("newrefresh")
		d.UserClient = &fakeUserClient{resp: resp}
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: expiredToken(t), Refresh: auth.Token{Raw: "r"}}}
		if got := resolveStart(d); got != nav.Home {
			t.Fatalf("got %v, want Home", got)
		}
	})

	t.Run("expired refresh ok but locked -> Unlock", func(t *testing.T) {
		d := baseDeps(t)
		d.Vault = vault.New()
		resp := &userv1.RefreshResponse{}
		resp.SetAccessToken("newaccess")
		resp.SetRefreshToken("newrefresh")
		d.UserClient = &fakeUserClient{resp: resp}
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: expiredToken(t), Refresh: auth.Token{Raw: "r"}}}
		if got := resolveStart(d); got != nav.Unlock {
			t.Fatalf("got %v, want Unlock", got)
		}
	})

	t.Run("expired refresh rpc error -> Welcome", func(t *testing.T) {
		d := baseDeps(t)
		d.UserClient = &fakeUserClient{err: errors.New("rpc down")}
		d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: expiredToken(t), Refresh: auth.Token{Raw: "r"}}}
		if got := resolveStart(d); got != nav.Welcome {
			t.Fatalf("got %v, want Welcome", got)
		}
	})

	t.Run("expired refresh save error -> Welcome", func(t *testing.T) {
		d := baseDeps(t)
		resp := &userv1.RefreshResponse{}
		resp.SetAccessToken("newaccess")
		resp.SetRefreshToken("newrefresh")
		d.UserClient = &fakeUserClient{resp: resp}
		d.SessionStore = &fakeStore{
			session: &auth.Session{Login: "a", Access: expiredToken(t), Refresh: auth.Token{Raw: "r"}},
			saveErr: errors.New("save fail"),
		}
		if got := resolveStart(d); got != nav.Welcome {
			t.Fatalf("got %v, want Welcome", got)
		}
	})
}

func TestNew(t *testing.T) {
	d := baseDeps(t)
	d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
	m := New(d)
	if m == nil {
		t.Fatal("New returned nil")
	}
	rm, ok := m.(rootModel)
	if !ok {
		t.Fatalf("New returned %T", m)
	}
	if rm.current == nil {
		t.Fatal("current model is nil")
	}
}

func TestBuildAllScreens(t *testing.T) {
	d := baseDeps(t)
	ids := []nav.ScreenID{
		nav.Welcome, nav.Login, nav.Register, nav.Home, nav.Unlock,
		nav.Cards, nav.CardAdd, nav.Passwords, nav.PasswordAdd,
		nav.Notes, nav.NoteAdd, nav.Files, nav.FileUpload,
		nav.Sync, nav.CardSync, nav.PasswordSync, nav.FileSync,
		nav.ScreenID(999),
	}
	for _, id := range ids {
		if got := build(d, id); got == nil {
			t.Fatalf("build(%v) returned nil", id)
		}
	}
}

func TestBuildUnlockGetError(t *testing.T) {
	d := baseDeps(t)
	d.SessionStore = &fakeStore{getErr: errors.New("boom")}
	if got := build(d, nav.Unlock); got == nil {
		t.Fatal("build Unlock with get error returned nil")
	}
}

func TestInit(t *testing.T) {
	d := baseDeps(t)
	d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
	m := New(d).(rootModel)
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
}

func newRoot(t *testing.T, d Deps) rootModel {
	t.Helper()
	d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
	return New(d).(rootModel)
}

func TestUpdatePushDataScreens(t *testing.T) {
	for _, id := range []nav.ScreenID{nav.Notes, nav.Cards, nav.Passwords, nav.Files} {
		d := baseDeps(t)
		m := newRoot(t, d)
		next, cmd := m.Update(nav.PushMsg{ID: id})
		nm := next.(rootModel)
		if len(nm.stack) != 1 {
			t.Fatalf("stack len = %d, want 1", len(nm.stack))
		}
		if cmd == nil {
			t.Fatalf("push %v returned nil cmd", id)
		}
	}
}

func TestUpdatePushNonSyncScreen(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)
	next, cmd := m.Update(nav.PushMsg{ID: nav.CardAdd})
	nm := next.(rootModel)
	if len(nm.stack) != 1 {
		t.Fatalf("stack len = %d, want 1", len(nm.stack))
	}
	_ = cmd
}

func TestUpdatePushModelAndBack(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)
	child := build(d, nav.Home)
	next, _ := m.Update(nav.PushModelMsg{Model: child})
	nm := next.(rootModel)
	if len(nm.stack) != 1 {
		t.Fatalf("stack len = %d, want 1", len(nm.stack))
	}

	back, _ := nm.Update(nav.BackMsg{})
	bm := back.(rootModel)
	if len(bm.stack) != 0 {
		t.Fatalf("stack len after back = %d, want 0", len(bm.stack))
	}
}

func TestUpdateBackEmptyStack(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)
	next, cmd := m.Update(nav.BackMsg{})
	if cmd != nil {
		t.Fatal("back on empty stack should return nil cmd")
	}
	if len(next.(rootModel).stack) != 0 {
		t.Fatal("stack should remain empty")
	}
}

func TestUpdateReset(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)
	m.stack = []tea.Model{build(d, nav.Home)}
	next, _ := m.Update(nav.ResetMsg{ID: nav.Home})
	nm := next.(rootModel)
	if nm.stack != nil {
		t.Fatalf("stack should be nil after reset, got %v", nm.stack)
	}
}

func TestUpdateLogout(t *testing.T) {
	d := baseDeps(t)
	store := &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
	d.SessionStore = store
	m := New(d).(rootModel)
	m.stack = []tea.Model{build(d, nav.Home)}
	next, _ := m.Update(nav.LogoutMsg{})
	nm := next.(rootModel)
	if !store.cleared {
		t.Fatal("session not cleared")
	}
	if !nm.Vault.Locked() {
		t.Fatal("vault should be locked after logout")
	}
	if nm.stack != nil {
		t.Fatal("stack should be nil after logout")
	}
}

func TestUpdateLogoutClearError(t *testing.T) {
	d := baseDeps(t)
	store := &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}, clearErr: errors.New("clear fail")}
	d.SessionStore = store
	m := New(d).(rootModel)
	next, cmd := m.Update(nav.LogoutMsg{})
	if cmd != nil {
		t.Fatal("logout with clear error should return nil cmd")
	}

	if next.(rootModel).Vault.Locked() {
		t.Fatal("vault should not be locked when clear failed")
	}
}

func TestUpdateSyncNow(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)
	_, cmd := m.Update(nav.SyncNowMsg{})
	if cmd == nil {
		t.Fatal("SyncNow should return a cmd")
	}
}

func TestUpdateSyncDoneFallthrough(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)

	next, cmd := m.Update(syncDoneMsg{err: errors.New("bad")})
	nm := next.(rootModel)
	if nm.syncErr != "bad" {
		t.Fatalf("syncErr = %q, want bad", nm.syncErr)
	}
	if cmd == nil {
		t.Fatal("syncDone should return reload cmd")
	}

	next2, _ := nm.Update(syncDoneMsg{offline: true})
	nm2 := next2.(rootModel)
	if nm2.syncErr != "" {
		t.Fatalf("syncErr should be cleared, got %q", nm2.syncErr)
	}
	if !nm2.offline {
		t.Fatal("offline should be set")
	}
}

func TestUpdateUnknownMsgDelegates(t *testing.T) {
	d := baseDeps(t)
	m := newRoot(t, d)

	next, _ := m.Update(struct{ x int }{1})
	if next == nil {
		t.Fatal("update returned nil model")
	}
}

func TestView(t *testing.T) {
	t.Run("locked vault no footer", func(t *testing.T) {
		d := baseDeps(t)
		d.Vault = vault.New()
		m := New(d).(rootModel)
		m.offline = true
		m.syncErr = "boom"
		m.conflicts = []conflictCount{{label: "Notes", n: 2}}
		v := m.View()
		if strings.Contains(v.Content, "offline") || strings.Contains(v.Content, "conflicts") || strings.Contains(v.Content, "Sync error") {
			t.Fatalf("locked vault must not render footer: %q", v.Content)
		}
	})

	t.Run("nil vault no footer", func(t *testing.T) {
		d := baseDeps(t)
		m := newRoot(t, d)
		m.Vault = nil
		m.offline = true
		v := m.View()
		if strings.Contains(v.Content, "offline") {
			t.Fatal("nil vault must not render footer")
		}
	})

	t.Run("unlocked with all footers", func(t *testing.T) {
		d := baseDeps(t)
		m := newRoot(t, d)
		m.offline = true
		m.syncErr = "network down"
		m.conflicts = []conflictCount{{label: "Notes", n: 3}, {label: "Cards", n: 1}}
		v := m.View()
		if !strings.Contains(v.Content, "offline mode") {
			t.Fatalf("missing offline footer: %q", v.Content)
		}
		if !strings.Contains(v.Content, "conflicts") || !strings.Contains(v.Content, "Notes: 3") || !strings.Contains(v.Content, "Cards: 1") {
			t.Fatalf("missing conflicts footer: %q", v.Content)
		}
		if !strings.Contains(v.Content, "Sync error: network down") {
			t.Fatalf("missing sync error footer: %q", v.Content)
		}
	})

	t.Run("unlocked no flags no footer", func(t *testing.T) {
		d := baseDeps(t)
		m := newRoot(t, d)
		v := m.View()
		if strings.Contains(v.Content, "offline") || strings.Contains(v.Content, "Sync error") {
			t.Fatalf("clean state must not render footer: %q", v.Content)
		}
	})
}

func TestIsOffline(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil not offline", nil, false},
		{"unavailable offline", status.Error(codes.Unavailable, "x"), true},
		{"deadline offline", status.Error(codes.DeadlineExceeded, "x"), true},
		{"internal not offline", status.Error(codes.Internal, "x"), false},
		{"plain error not offline", errors.New("x"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOffline(tt.err); got != tt.want {
				t.Fatalf("isOffline = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncOnEnter(t *testing.T) {
	on := []nav.ScreenID{nav.Notes, nav.Cards, nav.Passwords, nav.Files}
	for _, id := range on {
		if !syncOnEnter(id) {
			t.Fatalf("syncOnEnter(%v) = false, want true", id)
		}
	}
	off := []nav.ScreenID{nav.Welcome, nav.Home, nav.Login, nav.CardAdd, nav.NoteAdd}
	for _, id := range off {
		if syncOnEnter(id) {
			t.Fatalf("syncOnEnter(%v) = true, want false", id)
		}
	}
}

func TestConflictListers(t *testing.T) {
	t.Run("all repos present", func(t *testing.T) {
		d := baseDeps(t)
		m := New(setStartHome(t, d)).(rootModel)
		listers := m.conflictListers()
		if len(listers) != 4 {
			t.Fatalf("listers = %d, want 4", len(listers))
		}
	})

	t.Run("no repos", func(t *testing.T) {
		d := Deps{
			Vault:        testVault(t),
			Sync:         &fakeSyncer{},
			Logger:       &fakeLogger{},
			SessionStore: &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}},
		}
		m := New(d).(rootModel)
		if len(m.conflictListers()) != 0 {
			t.Fatal("expected no listers")
		}
	})
}

func setStartHome(t *testing.T, d Deps) Deps {
	t.Helper()
	d.SessionStore = &fakeStore{session: &auth.Session{Login: "a", Access: futureToken(t)}}
	return d
}

func TestSyncCmd(t *testing.T) {
	t.Run("nil syncer returns nil", func(t *testing.T) {
		d := baseDeps(t)
		d.Sync = nil
		m := newRoot(t, d)
		if cmd := m.syncCmd(); cmd != nil {
			t.Fatal("nil syncer should yield nil cmd")
		}
	})

	t.Run("success no conflicts", func(t *testing.T) {
		d := baseDeps(t)
		m := newRoot(t, d)
		cmd := m.syncCmd()
		if cmd == nil {
			t.Fatal("expected cmd")
		}
		msg := cmd().(syncDoneMsg)
		if msg.err != nil || msg.offline || len(msg.conflicts) != 0 {
			t.Fatalf("unexpected msg: %+v", msg)
		}
	})

	t.Run("sync offline error", func(t *testing.T) {
		d := baseDeps(t)
		d.Sync = &fakeSyncer{err: status.Error(codes.Unavailable, "down")}
		m := newRoot(t, d)
		msg := m.syncCmd()().(syncDoneMsg)
		if !msg.offline {
			t.Fatalf("expected offline, got %+v", msg)
		}
	})

	t.Run("sync generic error", func(t *testing.T) {
		d := baseDeps(t)
		d.Sync = &fakeSyncer{err: errors.New("boom")}
		log := &fakeLogger{}
		d.Logger = log
		m := newRoot(t, d)
		msg := m.syncCmd()().(syncDoneMsg)
		if msg.err == nil {
			t.Fatalf("expected error, got %+v", msg)
		}
		if log.errors == 0 {
			t.Fatal("logger should have recorded the error")
		}
	})

	t.Run("sync error with nil logger", func(t *testing.T) {
		d := baseDeps(t)
		d.Sync = &fakeSyncer{err: errors.New("boom")}
		d.Logger = nil
		m := newRoot(t, d)
		msg := m.syncCmd()().(syncDoneMsg)
		if msg.err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with conflicts reported", func(t *testing.T) {
		d := baseDeps(t)
		ctx := context.Background()

		n, err := d.NotesRepo.Create(ctx, []byte("v1"))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.NotesRepo.MarkConflict(ctx, n.ID, []byte("srv"), 5); err != nil {
			t.Fatal(err)
		}
		m := newRoot(t, d)
		msg := m.syncCmd()().(syncDoneMsg)
		if len(msg.conflicts) == 0 {
			t.Fatalf("expected conflicts, got %+v", msg)
		}
	})
}
