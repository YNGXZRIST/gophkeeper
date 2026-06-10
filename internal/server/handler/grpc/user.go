package grpc

import (
	"context"
	"fmt"
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
func (s *UserServer) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	login := in.GetLogin()
	password := in.GetPassword()
	fmt.Println(password, login)
	return &pb.RegisterResponse{}, nil
}
func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	login := in.GetLogin()
	password := in.GetPassword()
	fmt.Println(password, login)
	return &pb.LoginResponse{}, nil
}
