package transport

import (
	"context"
	"testing"
	"time"

	"gophkeeper/internal/server/service"

	"go.uber.org/zap"
)

type parserStub struct{}

func (parserStub) Parse(string) (string, error) { return "", nil }

func TestNewServerGRPC(t *testing.T) {
	srv, err := NewServer(ServerProp{
		Config:      &Config{Transport: GRPC, Address: "127.0.0.1:0"},
		Services:    &service.Services{},
		Logger:      zap.NewNop(),
		TokenParser: parserStub{},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if srv == nil {
		t.Fatal("NewServer() returned nil server")
	}

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestNewServerGRPCStartStop(t *testing.T) {
	srv, err := NewServer(ServerProp{
		Config:      &Config{Transport: GRPC, Address: "127.0.0.1:0"},
		Services:    &service.Services{},
		Logger:      zap.NewNop(),
		TokenParser: parserStub{},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	time.Sleep(50 * time.Millisecond)
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() returned error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}
}

func TestNewServerUnsupportedTransport(t *testing.T) {
	_, err := NewServer(ServerProp{
		Config:   &Config{Transport: HTTP, Address: "127.0.0.1:0"},
		Services: &service.Services{},
		Logger:   zap.NewNop(),
	})
	if err == nil {
		t.Fatal("NewServer() error = nil, want not-implemented error")
	}
}

func TestNewServerGRPCListenError(t *testing.T) {
	_, err := NewServer(ServerProp{
		Config:   &Config{Transport: GRPC, Address: "256.256.256.256:99999"},
		Services: &service.Services{},
		Logger:   zap.NewNop(),
	})
	if err == nil {
		t.Fatal("NewServer() error = nil, want listen error")
	}
}
