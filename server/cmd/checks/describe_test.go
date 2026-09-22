package checks_test

import (
	"strings"
	"testing"

	"server/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestDescribe(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsServerAddress := server.Addr().String()

	nc, err := nats.Connect(natsServerAddress)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
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

	t.Run("NoNats", func(t *testing.T) {
		err := checks.Describe("DNE", "", "noop")

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "failed to connect to NATS after 3 attempts: dial tcp: lookup DNE: no such host") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoCheck", func(t *testing.T) {
		err := checks.Describe(natsServerAddress, "", "DNE")

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "check does not exist") {
			t.Fatalf("Got wrong error message: %v", err.Error())
		}
	})

	t.Run("MalformedJson", func(t *testing.T) {
		_, err = kv.Put("check.bad", []byte("[}"))
		if err != nil {
			t.Fatal(err)
		}

		err := checks.Describe(natsServerAddress, "", "bad")

		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "invalid character '}' looking for beginning of value") {
			t.Fatalf("Got wrong error message: %v", err.Error())
		}
	})
}
