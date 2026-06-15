package auth

import (
	"context"
	userv1 "gophkeeper/internal/shared/proto/user/v1"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const authHeader = "authorization"

const refreshMinutes = 1

// SessionStore loads and persists the current session; satisfied by the
// session repository.
type SessionStore interface {
	Get(ctx context.Context) (*Session, error)
	Save(ctx context.Context, cred Credentials) (*Session, error)
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
func UnaryRefreshInterceptor(sessions SessionStore, log *zap.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if method != userv1.UserService_Refresh_FullMethodName {
			if session, err := sessions.Get(ctx); err == nil && session != nil && session.Refresh.Raw != "" {
				if time.Until(session.Access.ExpiresAt) <= refreshMinutes*time.Minute {
					in := &userv1.RefreshRequest{}
					in.SetRefreshToken(session.Refresh.Raw)
					resp, err := userv1.NewUserServiceClient(cc).Refresh(ctx, in)
					if err != nil {
						log.Error("refresh access token", zap.Error(err))
						return invoker(ctx, method, req, reply, cc, opts...)
					}
					if _, err := sessions.Save(ctx, Credentials{
						Login:        session.Login,
						AccessToken:  resp.GetAccessToken(),
						RefreshToken: resp.GetRefreshToken(),
						EncSalt:      session.EncSalt,
						WrappedDek:   session.WrappedDek,
					}); err != nil {
						log.Error("save refreshed session", zap.Error(err))
					}
				}
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
