package initialize_test

import (
	"testing"

	"server/cmd/initialize"

	natsserver "github.com/nats-io/nats-server/v2/test"
)

var NO_NATS_AUTH_FILE = ""

func TestUninitializedNATS(t *testing.T) {
	natsAddress := "nats://localhost:6001" // Non-default nats port

	err := initialize.Initialize(natsAddress, NO_NATS_AUTH_FILE)

	if err.Error() != "failed to connect to NATS: nats: no servers available for connection" {
		t.Fatal("Returned the incorrect error message")
	}
}

func TestConnection(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsAddress := server.Addr().String()

	err := initialize.Initialize(natsAddress, NO_NATS_AUTH_FILE)

	if err != nil {
		t.Fatal("Normal startup operations failed")
	}
}
