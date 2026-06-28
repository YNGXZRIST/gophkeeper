package transport

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/service"
	grpcServer "gophkeeper/internal/server/transport/grpc"
	"log/slog"
)

const (
	GRPC = "grpc"
	HTTP = "http"
)

// Server is the transport-agnostic contract for a running server.
type Server interface {
	Start() error
	Stop(ctx context.Context) error
}

type Config struct {
	Transport string
	Address   string
	CertFile  string
	KeyFile   string
}
type ServerProp struct {
	Config      *Config
	Services    *service.Services
	Logger      *slog.Logger
	TokenParser grpcServer.TokenParser
}

func NewServer(prop ServerProp) (Server, error) {
	switch prop.Config.Transport {
	case GRPC:
		return grpcServer.New(grpcServer.Deps{
			Address:     prop.Config.Address,
			CertFile:    prop.Config.CertFile,
			KeyFile:     prop.Config.KeyFile,
			Services:    prop.Services,
			Logger:      prop.Logger,
			TokenParser: prop.TokenParser,
		})
	default:
		return nil, fmt.Errorf("transport %q not yet implemented", prop.Config.Transport)
	}
}
