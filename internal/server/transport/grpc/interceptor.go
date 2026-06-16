package grpc

import (
	"context"
	"strings"

	"gophkeeper/internal/server/auth/authctx"
	pbU "gophkeeper/internal/shared/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TokenParser verifies an access token and returns its subject (user ID).
type TokenParser interface {
	Parse(token string) (string, error)
}

// publicMethods are the RPCs reachable without an access token.
var publicMethods = map[string]bool{
	pbU.UserService_Register_FullMethodName: true,
	pbU.UserService_Login_FullMethodName:    true,
	pbU.UserService_Refresh_FullMethodName:  true,
}

// AuthUnaryInterceptor authenticates each RPC and puts the user ID into the context.
func AuthUnaryInterceptor(parser TokenParser) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		token, err := bearerToken(ctx)
		if err != nil {
			return nil, err
		}
		userID, err := parser.Parse(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}
		return handler(authctx.ContextWithUserID(ctx, userID), req)
	}
}

func bearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}
	token, ok := strings.CutPrefix(values[0], "Bearer ")
	if !ok {
		return "", status.Error(codes.Unauthenticated, "authorization must be a bearer token")
	}
	return token, nil
}
