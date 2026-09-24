package nats_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/andrew-aiken/score/internal/auth"
	"github.com/andrew-aiken/score/internal/nats"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsnats "github.com/nats-io/nats.go"
)

func TestUser(t *testing.T) {
	username := "johndoe"

	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	nc, err := natsnats.Connect(server.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	bucket, err := js.CreateKeyValue(&natsnats.KeyValueConfig{
		Bucket:       "settings",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatal(err)
	}

	kv, err := js.KeyValue(bucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	// Insert random key to verify ListUsrs handles it correctly
	_, err = kv.Put("randomKey", []byte{})
	if err != nil {
		t.Fatalf("Failed to put random test key: %v", err)
	}

	t.Run("PutUser", func(t *testing.T) {
		err = nats.PutUser(kv, auth.User{
			Username:     username,
			Team:         "admin",
			PasswordHash: "fake",
		})

		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	})

	t.Run("GetUser", func(t *testing.T) {
		user, err := nats.GetUser(kv, username)
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
		}
		if user == (auth.User{}) {
			t.Fatal("Failed to find user")
		}
	})

	t.Run("UserExist", func(t *testing.T) {
		exist, err := nats.UserExists(kv, "dne")
		if err != nil {
			t.Fatalf("Failed to retrieve user: %v", err)
		}
		if exist {
			t.Fatal("User should not exist")
		}
	})

	t.Run("ListUsers", func(t *testing.T) {
		users, err := nats.ListUsers(kv)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}

		// This will fail if running only this test
		if users[0].Username != username {
			t.Fatal("Username value does not match")
		}
	})

	// Tests the edge case of a user with json that does not follow auth.User (server/internal/auth)
	t.Run("ListUsersBadUser", func(t *testing.T) {
		// Insert badly formatted user value
		_, err = kv.Put("user.baduser", []byte("foo"))
		if err != nil {
			t.Fatalf("Failed to put bad user: %v", err)
		}

		_, err := nats.ListUsers(kv)
		unwrappedError := errors.Unwrap(err)
		if _, ok := errors.AsType[*json.SyntaxError](unwrappedError); !ok {
			t.Fatalf("Got incorrect error when unwrapped: %T", unwrappedError)
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		err := nats.DeleteUser(kv, username)
		if err != nil {
			t.Fatalf("Failed to delete users: %v", err)
		}
	})
}
