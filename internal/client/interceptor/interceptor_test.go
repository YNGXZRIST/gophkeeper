package interceptor

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"gophkeeper/internal/client/auth"
	userv1 "gophkeeper/internal/shared/proto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type fakeStore struct {
	session   *auth.Session
	getErr    error
	saved     []auth.Credentials
	saveErr   error
	saveCalls int
}

func (f *fakeStore) Get(context.Context) (*auth.Session, error) {
	return f.session, f.getErr
}

func (f *fakeStore) Save(_ context.Context, cred auth.Credentials) (*auth.Session, error) {
	f.saveCalls++
	f.saved = append(f.saved, cred)
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	s := auth.NewSession(cred.Login,
		auth.Token{Raw: cred.AccessToken},
		auth.Token{Raw: cred.RefreshToken},
		cred.EncSalt, cred.WrappedDek)
	f.session = s
	return s, nil
}

type fakeUserServer struct {
	userv1.UnimplementedUserServiceServer
	mu       sync.Mutex
	calls    int
	gotToken string
	respAcc  string
	respRef  string
	err      error
}

func (s *fakeUserServer) Refresh(_ context.Context, in *userv1.RefreshRequest) (*userv1.RefreshResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.gotToken = in.GetRefreshToken()
	if s.err != nil {
		return nil, s.err
	}
	out := &userv1.RefreshResponse{}
	out.SetAccessToken(s.respAcc)
	out.SetRefreshToken(s.respRef)
	return out, nil
}

func newConn(t *testing.T, srv *fakeUserServer) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	userv1.RegisterUserServiceServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
		gs.Stop()
	})
	return conn
}

func sessionWith(access string, exp time.Time, refresh string) *auth.Session {
	return auth.NewSession("user",
		auth.Token{Raw: access, ExpiresAt: exp},
		auth.Token{Raw: refresh, ExpiresAt: time.Now().Add(24 * time.Hour)},
		[]byte("salt"), []byte("dek"))
}

func headerFrom(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	vals := md.Get(authHeader)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func TestUnaryAuthInterceptor_AttachesToken(t *testing.T) {
	store := &fakeStore{session: sessionWith("acc123", time.Now().Add(time.Hour), "ref")}
	var seen string
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		seen = headerFrom(ctx)
		return nil
	}

	err := UnaryAuthInterceptor(store)(context.Background(), "/svc/M", nil, nil, nil, invoker)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if seen != "Bearer acc123" {
		t.Fatalf("header = %q, want Bearer acc123", seen)
	}
}

func TestUnaryAuthInterceptor_NoSession(t *testing.T) {
	store := &fakeStore{session: nil}
	var seen = "unset"
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		seen = headerFrom(ctx)
		return nil
	}
	if err := UnaryAuthInterceptor(store)(context.Background(), "/svc/M", nil, nil, nil, invoker); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if seen != "" {
		t.Fatalf("expected no auth header, got %q", seen)
	}
}

func TestStreamAuthInterceptor_AttachesToken(t *testing.T) {
	store := &fakeStore{session: sessionWith("streamtok", time.Now().Add(time.Hour), "ref")}
	var seen string
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		seen = headerFrom(ctx)
		return nil, nil
	}
	_, err := StreamAuthInterceptor(store)(context.Background(), &grpc.StreamDesc{}, nil, "/svc/Stream", streamer)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if seen != "Bearer streamtok" {
		t.Fatalf("header = %q", seen)
	}
}

func TestUnaryRefreshInterceptor_RefreshesNearExpiry(t *testing.T) {
	srv := &fakeUserServer{respAcc: "new-access", respRef: "new-refresh"}
	conn := newConn(t, srv)

	store := &fakeStore{session: sessionWith("old-access", time.Now().Add(30*time.Second), "old-refresh")}

	var invoked bool
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		invoked = true
		return nil
	}

	err := UnaryRefreshInterceptor(store, zap.NewNop())(
		context.Background(), "/card.v1.CardService/Add", nil, nil, conn, invoker)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if srv.calls != 1 {
		t.Fatalf("Refresh calls = %d, want 1", srv.calls)
	}
	if srv.gotToken != "old-refresh" {
		t.Fatalf("refresh token sent = %q", srv.gotToken)
	}
	if store.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1", store.saveCalls)
	}
	if store.saved[0].AccessToken != "new-access" || store.saved[0].RefreshToken != "new-refresh" {
		t.Fatalf("saved creds = %+v", store.saved[0])
	}
	if !invoked {
		t.Fatal("invoker not called")
	}
}

func TestUnaryRefreshInterceptor_SkipsWhenNotNearExpiry(t *testing.T) {
	srv := &fakeUserServer{}
	conn := newConn(t, srv)
	store := &fakeStore{session: sessionWith("acc", time.Now().Add(time.Hour), "ref")}

	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error { return nil }
	if err := UnaryRefreshInterceptor(store, zap.NewNop())(
		context.Background(), "/svc/M", nil, nil, conn, invoker); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if srv.calls != 0 {
		t.Fatalf("expected no refresh, calls = %d", srv.calls)
	}
}

func TestUnaryRefreshInterceptor_SkipsRefreshMethod(t *testing.T) {
	srv := &fakeUserServer{}
	conn := newConn(t, srv)
	store := &fakeStore{session: sessionWith("acc", time.Now().Add(time.Second), "ref")}

	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error { return nil }
	if err := UnaryRefreshInterceptor(store, zap.NewNop())(
		context.Background(), userv1.UserService_Refresh_FullMethodName, nil, nil, conn, invoker); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if srv.calls != 0 {
		t.Fatalf("Refresh method must not trigger nested refresh, calls = %d", srv.calls)
	}
}

func TestUnaryRefreshInterceptor_RefreshError(t *testing.T) {
	srv := &fakeUserServer{err: context.DeadlineExceeded}
	conn := newConn(t, srv)
	store := &fakeStore{session: sessionWith("acc", time.Now().Add(time.Second), "ref")}

	var invoked bool
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		invoked = true
		return nil
	}
	if err := UnaryRefreshInterceptor(store, zap.NewNop())(
		context.Background(), "/svc/M", nil, nil, conn, invoker); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("Save must not be called on refresh error, calls = %d", store.saveCalls)
	}
	if !invoked {
		t.Fatal("invoker must still run after refresh error")
	}
}

func TestUnaryRefreshInterceptor_NoSession(t *testing.T) {
	srv := &fakeUserServer{}
	conn := newConn(t, srv)
	store := &fakeStore{session: nil}

	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error { return nil }
	if err := UnaryRefreshInterceptor(store, zap.NewNop())(
		context.Background(), "/svc/M", nil, nil, conn, invoker); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if srv.calls != 0 {
		t.Fatalf("no session must not refresh, calls = %d", srv.calls)
	}
}

func TestStreamRefreshInterceptor_RefreshesNearExpiry(t *testing.T) {
	srv := &fakeUserServer{respAcc: "sa", respRef: "sr"}
	conn := newConn(t, srv)
	store := &fakeStore{session: sessionWith("acc", time.Now().Add(10*time.Second), "ref")}

	var opened bool
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		opened = true
		return nil, nil
	}
	_, err := StreamRefreshInterceptor(store, zap.NewNop())(
		context.Background(), &grpc.StreamDesc{}, conn, "/svc/Stream", streamer)
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if srv.calls != 1 {
		t.Fatalf("Refresh calls = %d, want 1", srv.calls)
	}
	if store.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1", store.saveCalls)
	}
	if !opened {
		t.Fatal("stream not opened")
	}
}

func TestStreamRefreshInterceptor_Error(t *testing.T) {
	srv := &fakeUserServer{err: context.DeadlineExceeded}
	conn := newConn(t, srv)
	store := &fakeStore{session: sessionWith("acc", time.Now().Add(time.Second), "ref")}

	var opened bool
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		opened = true
		return nil, nil
	}
	if _, err := StreamRefreshInterceptor(store, zap.NewNop())(
		context.Background(), &grpc.StreamDesc{}, conn, "/svc/Stream", streamer); err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("Save must not run on refresh error")
	}
	if !opened {
		t.Fatal("stream must still open after refresh error")
	}
}
