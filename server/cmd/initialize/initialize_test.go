package initialize_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"server/cmd/initialize"
	"server/internal/config"

	natsserver "github.com/nats-io/nats-server/v2/test"
)

func TestMissingConfigFile(t *testing.T) {
	err := initialize.Initialize("config.json")

	_, ok := errors.AsType[*fs.PathError](err)
	if !ok {
		t.FailNow()
	}
}

func TestUninitializedNATS(t *testing.T) {
	configFile := testGenerateConfigFile(t, config.Config{
		NATSUrl:       "nats://localhost:6001", // Non-default nats port
		NATSCredsFile: "",
	})

	err := initialize.Initialize(configFile)

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

	configFile := testGenerateConfigFile(t, config.Config{
		NATSUrl:       server.Addr().String(),
		NATSCredsFile: "",
	})

	err := initialize.Initialize(configFile)

	if err != nil {
		t.Fatal("Normal startup operations failed")
	}
}

func testGenerateConfigFile(t *testing.T, config config.Config) string {
	tmpDir := t.TempDir()

	tmpConfigFile := filepath.Join(tmpDir, "config.json")

	content, err := json.Marshal(config)
	if err != nil {
		t.FailNow()
	}

	if err := os.WriteFile(tmpConfigFile, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	return tmpConfigFile
}
