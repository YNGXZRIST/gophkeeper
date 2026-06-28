package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/user/v1"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	UserServerProp
}
type UserServerProp struct {
	Service *service.UserService
	Logger  *slog.Logger
}

func NewUserServer(prop UserServerProp) *UserServer {
	return &UserServer{UserServerProp: prop}
}
func (s *UserServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, tokens, err := s.Service.Register(ctx, model.User{Login: in.GetLogin(), Pass: in.GetPassword(), WrappedDesk: in.GetWrappedDek(), EncSalt: in.GetEncSalt()})
	if err != nil {
		if errors.Is(err, model.ErrLoginTaken) {
			return nil, status.Error(codes.AlreadyExists, "user already registered")
		}
		s.Logger.Error("register failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbUser := &pb.User{}
	pbUser.SetId(user.ID)
	pbUser.SetLogin(user.Login)
	resp := &pb.RegisterResponse{}
	resp.SetUser(pbUser)
	resp.SetAccessToken(tokens.Access)
	resp.SetRefreshToken(tokens.Refresh)
	return resp, nil
}
func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, tokens, err := s.Service.Login(ctx, in.GetLogin(), in.GetPassword())
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid login or password")
		}
		s.Logger.Error("login failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	pbUser := &pb.User{}
	pbUser.SetId(user.ID)
	pbUser.SetLogin(user.Login)
	resp := &pb.LoginResponse{}
	resp.SetUser(pbUser)
	resp.SetAccessToken(tokens.Access)
	resp.SetRefreshToken(tokens.Refresh)
	resp.SetEncSalt(user.EncSalt)
	resp.SetWrappedDek(user.WrappedDesk)
	return resp, nil
}

func (s *UserServer) Refresh(ctx context.Context, in *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	tokens, err := s.Service.Refresh(ctx, in.GetRefreshToken())
	if err != nil {
		if errors.Is(err, model.ErrInvalidRefreshToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		s.Logger.Error("refresh failed", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	resp := &pb.RefreshResponse{}
	resp.SetAccessToken(tokens.Access)
	resp.SetRefreshToken(tokens.Refresh)
	return resp, nil
}
