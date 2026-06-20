package grpc

import (
	"context"
	"fmt"
	handler "gophkeeper/internal/server/handler/grpc"
	"gophkeeper/internal/server/service"
	pbC "gophkeeper/internal/shared/proto/card/v1"
	pbF "gophkeeper/internal/shared/proto/file/v1"
	pbN "gophkeeper/internal/shared/proto/note/v1"
	pbP "gophkeeper/internal/shared/proto/password/v1"
	pbU "gophkeeper/internal/shared/proto/user/v1"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Deps are the dependencies the gRPC server needs to register its handlers.
type Deps struct {
	Address     string
	Services    *service.Services
	Logger      *zap.Logger
	TokenParser TokenParser
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

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(AuthUnaryInterceptor(d.TokenParser)),
		grpc.ChainStreamInterceptor(AuthStreamInterceptor(d.TokenParser)),
	)
	pbU.RegisterUserServiceServer(grpcSrv, handler.NewUserServer(handler.UserServerProp{
		Service: d.Services.User,
		Logger:  d.Logger,
	}))
	pbC.RegisterCardServiceServer(grpcSrv, handler.NewCardServer(handler.CardServerProp{
		Service: d.Services.Card,
		Logger:  d.Logger,
	}))
	pbP.RegisterPasswordServiceServer(grpcSrv, handler.NewPasswordServer(handler.PasswordServerProp{
		Service: d.Services.Password,
		Logger:  d.Logger,
	}))
	pbN.RegisterNoteServiceServer(grpcSrv, handler.NewNoteServer(handler.NoteServerProp{
		Service: d.Services.Note,
		Logger:  d.Logger,
	}))
	pbF.RegisterFileServiceServer(grpcSrv, handler.NewFileServer(handler.FileServerProp{
		Service: d.Services.File,
		Logger:  d.Logger,
	}))

	return &Server{srv: grpcSrv, lis: lis}, nil
}
