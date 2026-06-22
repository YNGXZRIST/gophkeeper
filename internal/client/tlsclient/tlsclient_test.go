package tlsclient_test

import (
	"context"
	"net"
	"testing"
	"time"

	"gophkeeper/internal/client/tlsclient"
	"gophkeeper/internal/server/tlsserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCredentials(t *testing.T) {
	creds, err := tlsclient.Credentials()
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if creds == nil {
		t.Fatalf("Credentials returned nil")
	}
}

// TestHandshakeAgainstServer dials a gRPC server configured with the repo's
// server cert+key (the same pair the client embeds) and invokes a bogus method.
// A codes.Unimplemented reply proves the TLS handshake succeeded.
func TestHandshakeAgainstServer(t *testing.T) {
	serverCreds, err := tlsserver.LoadCredentials("../../../certs/server.crt", "../../../certs/server.key")
	if err != nil {
		t.Skipf("server credentials unavailable (run make certs): %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.Creds(serverCreds))
	go func() {
		// Serve returns when the listener is closed at test teardown.
		if serveErr := srv.Serve(lis); serveErr != nil {
			t.Logf("server stopped: %v", serveErr)
		}
	}()
	t.Cleanup(srv.Stop)

	clientCreds, err := tlsclient.Credentials()
	if err != nil {
		t.Fatalf("client Credentials: %v", err)
	}

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(clientCreds))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() {
		if cerr := conn.Close(); cerr != nil {
			t.Logf("close conn: %v", cerr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = conn.Invoke(ctx, "/bogus.Service/Method", &emptyMsg{}, &emptyMsg{})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unimplemented {
		t.Fatalf("expected Unimplemented (handshake OK), got %s: %v", st.Code(), err)
	}
}

// emptyMsg is a minimal proto.Message-compatible payload for the bogus invoke.
type emptyMsg struct{}

func (m *emptyMsg) Reset()         {}
func (m *emptyMsg) String() string { return "" }
func (m *emptyMsg) ProtoMessage()  {}
