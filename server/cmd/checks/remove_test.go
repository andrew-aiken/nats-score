package checks_test

import (
	"strings"
	"testing"

	"github.com/andrew-aiken/score/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestRemove(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsServerAddress := server.Addr().String()
	checkName := "noop"

	nc, err := nats.Connect(natsServerAddress)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
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
		err := checks.Remove("DNE", "", checkName)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "failed to connect to NATS after 3 attempts: dial tcp: lookup DNE") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoCheck", func(t *testing.T) {
		err := checks.Remove(natsServerAddress, "", checkName)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "check does not exist") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})
}
