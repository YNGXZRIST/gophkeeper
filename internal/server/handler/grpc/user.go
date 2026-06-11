package grpc

import (
	"context"
	"errors"
	"gophkeeper/internal/server/model"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/user/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	UserServerProp
}
type UserServerProp struct {
	Service *service.UserService
	Logger  *zap.Logger
}

func NewUserServer(prop UserServerProp) *UserServer {
	return &UserServer{UserServerProp: prop}
}
func (s *UserServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, tokens, err := s.Service.Register(ctx, model.User{Login: in.GetLogin(), Pass: in.GetPassword()})
	if err != nil {
		if errors.Is(err, model.ErrLoginTaken) {
			return nil, status.Error(codes.AlreadyExists, "user already registered")
		}
		s.Logger.Error("register failed", zap.Error(err))
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
	return &pb.LoginResponse{}, nil
}
