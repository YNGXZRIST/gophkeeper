package grpc

import (
	"context"
	"fmt"
	handler "gophkeeper/internal/server/handler/grpc"
	"gophkeeper/internal/server/service"
	pb "gophkeeper/internal/shared/proto/user/v1"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Deps are the dependencies the gRPC server needs to register its handlers.
type Deps struct {
	Address  string
	Services *service.Services
	Logger   *zap.Logger
}

type Server struct {
	srv *grpc.Server
	lis net.Listener
}

func (s *Server) Start() error {
	return s.srv.Serve(s.lis)
}
func (s *Server) Stop(ctx context.Context) error {
	s.srv.GracefulStop()
	return nil
}
func New(d Deps) (*Server, error) {
	lis, err := net.Listen("tcp", d.Address)
	if err != nil {
		return nil, fmt.Errorf("listen %q: %w", d.Address, err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcSrv, handler.NewUserServer(handler.UserServerProp{
		Service: d.Services.User,
		Logger:  d.Logger,
	}))

	return &Server{srv: grpcSrv, lis: lis}, nil
}
