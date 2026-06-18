package grpc

import (
	"context"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/password/v1"

	"go.uber.org/zap"
)

type PasswordServer struct {
	pb.UnimplementedPasswordServiceServer
	PasswordServerProp
}

func (p *PasswordServer) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PasswordServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PasswordServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PasswordServer) Update(ctx context.Context, in *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PasswordServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	//TODO implement me
	panic("implement me")
}

type PasswordServerProp struct {
	Service *service.PasswordService
	Logger  *zap.Logger
}

func NewPasswordServer(passwordServerProp PasswordServerProp) *PasswordServer {
	return &PasswordServer{PasswordServerProp: passwordServerProp}
}
