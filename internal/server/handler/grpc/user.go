package grpc

import (
	"context"
	"gophkeeper/internal/server/model"
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
	user, tokens, err := s.Service.Register(ctx, model.User{Login: in.GetLogin(), Pass: in.GetPassword()})
	if err != nil {
		return nil, err
	}
	pbUser := &pb.User{}
	pbUser.SetId(user.ID)
	pbUser.SetLogin(user.Login)
	resp := &pb.RegisterResponse{}
	resp.SetUser(pbUser)
	resp.SetAccessToken(tokens.Access)
	return resp, nil
}
func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{}, nil
}
