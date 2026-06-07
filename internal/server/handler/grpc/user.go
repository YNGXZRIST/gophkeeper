package grpc

import (
	"context"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/user/v1"

	"go.uber.org/zap"
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
func (s *UserServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{}, nil
}
func (s *UserServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{}, nil
}
