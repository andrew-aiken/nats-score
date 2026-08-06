package nats_test

import (
	"context"
	"testing"
	"time"

	"server/internal/nats"
	"server/internal/settings"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsnats "github.com/nats-io/nats.go"
)

func TestNats(t *testing.T) {

	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsServerAddress := server.Addr().String()

	nc, err := natsnats.Connect(natsServerAddress)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	settingsBucket, err := js.CreateKeyValue(&natsnats.KeyValueConfig{
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

	_, err = js.CreateKeyValue(&natsnats.KeyValueConfig{
		Bucket:       "users",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatal(err)
	}

	kv, err := js.KeyValue(settingsBucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	testKeyName := "test.key"
	_, err = kv.Put(testKeyName, []byte{})
	if err != nil {
		t.Fatalf("Failed to put random test key: %v", err)
	}

	natsConn := nats.NatsConnection{
		NatsUrl:            natsServerAddress,
		NatsConnectionName: "test",
		NatsCredsFile:      "", // TODO see if its possible to implement
		NatsInboxPrefix:    "_INBOX",
	}
	defer natsConn.Close()

	t.Run("SetupConnection", func(t *testing.T) {
		err = natsConn.SetupConnection()
		if err != nil {
			t.Fatalf("Failed to setup NATS connection: %v", err)
		}
	})

	t.Run("SetupUsersKV", func(t *testing.T) {
		err = natsConn.SetupUsersKV()
		if err != nil {
			t.Fatalf("Failed to get users: %v", err)
		}
	})

	t.Run("SetupKVWatcher", func(t *testing.T) {
		err = natsConn.SetupKVWatcher([]string{testKeyName})
		if err != nil {
			t.Fatalf("Failed to watch keys %v", err)
		}
	})

	t.Run("SubjectSubscribe", func(t *testing.T) {
		ctx := context.Background()
		defer ctx.Done()

		settings := settings.Settings{}

		err = natsConn.SubjectSubscribe(ctx, &settings)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
}
