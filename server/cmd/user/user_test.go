package user_test

import (
	"testing"

	"github.com/andrew-aiken/score/cmd/initialize"
	"github.com/andrew-aiken/score/cmd/user"

	natsserver "github.com/nats-io/nats-server/v2/test"
)

var NO_NATS_AUTH_FILE = ""

func TestUserLifeCycle(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	testUser := "testUser"
	serverAddress := server.Addr().String()

	err := initialize.Initialize(serverAddress, NO_NATS_AUTH_FILE)
	if err != nil {
		t.Error("Failed to initialize nats setup")
	}

	err = user.Add(serverAddress, NO_NATS_AUTH_FILE, testUser, "admin", "testPassword", false)
	if err != nil {
		t.Error("Failed to create user")
	}

	err = user.List(serverAddress, NO_NATS_AUTH_FILE)
	if err != nil {
		t.Error("Failed to list users")
	}

	err = user.Remove(serverAddress, NO_NATS_AUTH_FILE, testUser)
	if err != nil {
		t.Error("Failed to remove users")
	}
}

func TestAddUser(t *testing.T) {
	t.Run("Missing username", func(t *testing.T) {
		err := user.Add("dne", NO_NATS_AUTH_FILE, "", "admin", "testPassword", false)

		if err.Error() != "username must not be empty" {
			t.Fatal(err)
		}
	})

	t.Run("Missing team", func(t *testing.T) {
		err := user.Add("dne", NO_NATS_AUTH_FILE, "admin", "", "testPassword", false)

		if err.Error() != "team must not be empty" {
			t.Fatal(err)
		}
	})

	t.Run("Missing password", func(t *testing.T) {
		err := user.Add("dne", NO_NATS_AUTH_FILE, "admin", "admin", "", false)

		if err.Error() != "password must not be empty" {
			t.Fatal(err)
		}
	})
}
