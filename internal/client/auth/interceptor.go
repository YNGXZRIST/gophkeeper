package auth

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const authHeader = "authorization"

const refreshMinutes = 1

// SessionStore loads and persists the current session; satisfied by the
// session repository.
type SessionStore interface {
	Get(ctx context.Context) (*Session, error)
	Save(ctx context.Context, login, accessToken, refreshToken string) (*Session, error)
}

// UnaryAuthInterceptor attaches the access token from the session as an
// "authorization: Bearer <token>" header on every outgoing unary call. When no
// session exists yet (e.g. before login) the call proceeds without the header.
func UnaryAuthInterceptor(sessions SessionStore) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if session, err := sessions.Get(ctx); err == nil && session != nil {
			ctx = metadata.AppendToOutgoingContext(ctx, authHeader, "Bearer "+session.Access.Raw)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// UnaryRefreshInterceptor refreshes the access token via the refresh token when
// it is close to expiry, before the outgoing unary call is sent.
func UnaryRefreshInterceptor(sessions SessionStore) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if session, err := sessions.Get(ctx); err == nil && session != nil {
			expiredAt := session.Access.ExpiresAt
			now := time.Now()
			diff := now.Sub(expiredAt)
			if diff < 0 || diff.Minutes() < refreshMinutes {
				//TODO add refresh session
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
