package transport

import (
	"context"
	"fmt"
	"gophkeeper/internal/server/service"
	grpcServer "gophkeeper/internal/server/transport/grpc"

	"go.uber.org/zap"
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
}
type ServerProp struct {
	Config   *Config
	Services *service.Services
	Logger   *zap.Logger
}

func NewServer(prop ServerProp) (Server, error) {
	switch prop.Config.Transport {
	case GRPC:
		return grpcServer.New(grpcServer.Deps{
			Address:  prop.Config.Address,
			Services: prop.Services,
			Logger:   prop.Logger,
		})
	default:
		return nil, fmt.Errorf("transport %q not yet implemented", prop.Config.Transport)
	}
}
