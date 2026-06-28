package grpc

import (
	"context"
	"errors"
	"testing"

	"gophkeeper/internal/server/auth/authctx"
	pbU "gophkeeper/internal/shared/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type parserStub struct {
	fn func(string) (string, error)
}

func (p parserStub) Parse(token string) (string, error) { return p.fn(token) }

func TestAuthUnaryInterceptor(t *testing.T) {
	const protected = "/card.v1.CardService/Add"

	tests := []struct {
		name       string
		method     string
		ctx        context.Context
		parse      func(string) (string, error)
		wantCode   codes.Code
		wantUserID string // expected uid seen by the handler ("" = handler must not run)
	}{
		{
			name:       "public method skips auth",
			method:     pbU.UserService_Login_FullMethodName,
			ctx:        context.Background(),
			wantUserID: "",
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
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic abc")),
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
			parse:      func(string) (string, error) { return "u1", nil },
			wantUserID: "u1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerRan bool
			var seenUserID string
			handler := func(ctx context.Context, _ any) (any, error) {
				handlerRan = true
				seenUserID, _ = authctx.UserIDFromContext(ctx)
				return "ok", nil
			}

			interceptor := AuthUnaryInterceptor(parserStub{fn: tt.parse})
			_, err := interceptor(tt.ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, handler)

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
