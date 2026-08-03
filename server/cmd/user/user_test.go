package user_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"server/cmd/initialize"
	"server/cmd/user"
	"server/internal/config"

	natsserver "github.com/nats-io/nats-server/v2/test"
)

func TestUserLifeCycle(t *testing.T) {
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

	initialize.Initialize(configFile)

	testUser := "testUser"

	err := user.Add(configFile, testUser, "admin", "testPassword", false)
	if err != nil {
		t.Error("Failed to create user")
	}

	err = user.List(configFile)
	if err != nil {
		t.Error("Failed to list users")
	}

	err = user.Remove(configFile, testUser)
	if err != nil {
		t.Error("Failed to remove users")
	}
}

func TestAddUser(t *testing.T) {
	t.Run("Missing username", func(t *testing.T) {
		err := user.Add("dne", "", "admin", "testPassword", false)

		if err.Error() != "username must not be empty" {
			t.Fatal(err)
		}
	})

	t.Run("Missing team", func(t *testing.T) {
		err := user.Add("dne", "admin", "", "testPassword", false)

		if err.Error() != "team must not be empty" {
			t.Fatal(err)
		}
	})

	t.Run("Missing password", func(t *testing.T) {
		err := user.Add("dne", "admin", "admin", "", false)

		if err.Error() != "password must not be empty" {
			t.Fatal(err)
		}
	})

	t.Run("Invalid config", func(t *testing.T) {
		configFileName := "bad-config.json"

		err := user.Add(configFileName, "admin", "admin", "password", false)

		if err.Error() != "open "+configFileName+": no such file or directory" {
			t.Fatal(err)
		}
	})
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
