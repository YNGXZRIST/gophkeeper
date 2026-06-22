package grpc

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gophkeeper/internal/server/auth/authctx"
	"gophkeeper/internal/server/service"
	"gophkeeper/internal/shared/certgen"
	pbU "gophkeeper/internal/shared/proto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func tempCerts(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	certPEM, keyPEM, err := certgen.Generate(certgen.DefaultOptions())
	if err != nil {
		t.Fatalf("generate certs: %v", err)
	}
	if err := certgen.WriteFiles(certPath, keyPath, certPEM, keyPEM); err != nil {
		t.Fatalf("write certs: %v", err)
	}
	return certPath, keyPath
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeStream) Context() context.Context { return s.ctx }

func TestAuthStreamInterceptor(t *testing.T) {
	const protected = "/file.v1.FileService/Download"

	tests := []struct {
		name       string
		method     string
		ctx        context.Context
		parse      func(string) (string, error)
		wantCode   codes.Code
		wantUserID string
	}{
		{
			name:   "public method skips auth",
			method: pbU.UserService_Register_FullMethodName,
			ctx:    context.Background(),
		},
		{
			name:     "missing metadata",
			method:   protected,
			ctx:      context.Background(),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "missing authorization header",
			method:   protected,
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "not a bearer token",
			method:   protected,
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic x")),
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid token",
			method:   protected,
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer bad")),
			parse:    func(string) (string, error) { return "", errors.New("invalid") },
			wantCode: codes.Unauthenticated,
		},
		{
			name:       "success injects user id",
			method:     protected,
			ctx:        metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer good")),
			parse:      func(string) (string, error) { return "u9", nil },
			wantUserID: "u9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerRan bool
			var seenUserID string
			handler := func(_ any, ss grpc.ServerStream) error {
				handlerRan = true
				seenUserID, _ = authctx.UserIDFromContext(ss.Context())
				return nil
			}

			interceptor := AuthStreamInterceptor(parserStub{fn: tt.parse})
			ss := &fakeStream{ctx: tt.ctx}
			err := interceptor(nil, ss, &grpc.StreamServerInfo{FullMethod: tt.method}, handler)

			if tt.wantCode != codes.OK {
				if status.Code(err) != tt.wantCode {
					t.Fatalf("code = %v, want %v (err=%v)", status.Code(err), tt.wantCode, err)
				}
				if handlerRan {
					t.Fatal("handler ran despite auth failure")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !handlerRan {
				t.Fatal("handler did not run")
			}
			if seenUserID != tt.wantUserID {
				t.Fatalf("handler saw uid %q, want %q", seenUserID, tt.wantUserID)
			}
		})
	}
}

func TestAuthStreamContext(t *testing.T) {
	want := context.WithValue(context.Background(), struct{}{}, "v")
	s := &authStream{ctx: want}
	if s.Context() != want {
		t.Fatal("authStream.Context() did not return injected context")
	}
}

func TestNewRegistersAndServes(t *testing.T) {
	cert, key := tempCerts(t)
	srv, err := New(Deps{
		Address:     "127.0.0.1:0",
		CertFile:    cert,
		KeyFile:     key,
		Services:    &service.Services{},
		Logger:      zap.NewNop(),
		TokenParser: parserStub{fn: func(string) (string, error) { return "", nil }},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	time.Sleep(50 * time.Millisecond)
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}
}

func TestNewListenError(t *testing.T) {
	cert, key := tempCerts(t)
	_, err := New(Deps{
		Address:     "256.256.256.256:99999",
		CertFile:    cert,
		KeyFile:     key,
		Services:    &service.Services{},
		Logger:      zap.NewNop(),
		TokenParser: parserStub{},
	})
	if err == nil {
		t.Fatal("New() error = nil, want listen error")
	}
}
