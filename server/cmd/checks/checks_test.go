package checks_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/andrew-aiken/score/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func Test(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsServerAddress := server.Addr().String()
	checksDir := "./testdata/"
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

	t.Run("Import", func(t *testing.T) {
		err := checks.Import(natsServerAddress, "", checksDir)
		if err != nil {
			t.Fatal(err.Error())
		}

		keys, err := kv.Keys()
		if err != nil {
			t.Fatal(err.Error())
		}
		if len(keys) != 1 {
			t.Errorf("One check should exist, got %d", len(keys))
		}

		t.Run("List", func(t *testing.T) {
			err = checks.List(natsServerAddress, "")
			if err != nil {
				t.Fatal(err.Error())
			}
		})

		t.Run("Describe", func(t *testing.T) {
			err = checks.Describe(natsServerAddress, "", checkName)
			if err != nil {
				t.Fatal(err.Error())
			}
		})

		t.Run("Validate", func(t *testing.T) {
			err := checks.Validate(natsServerAddress, "", checkName)

			if err != nil {
				t.Fatal(err.Error())
			}
		})
	})

	t.Run("Invalid Dir", func(t *testing.T) {
		err := checks.Import(natsServerAddress, "", "./dne")

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "stat ./dne: no such file or directory") {
			t.Fatalf("Got wrong error message: %v", err.Error())
		}
	})

	testExportDir := t.TempDir()

	t.Run("Export", func(t *testing.T) {
		err := checks.Export(natsServerAddress, "", testExportDir)

		if err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("Purge", func(t *testing.T) {
		err := checks.Purge(natsServerAddress, "", true)

		if err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("Import the Export", func(t *testing.T) {
		err := checks.Import(natsServerAddress, "", testExportDir)

		if err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("Remove Check", func(t *testing.T) {
		err := checks.Import(natsServerAddress, "", checksDir)
		if err != nil {
			t.Fatal(err.Error())
		}

		err = checks.Remove(natsServerAddress, "", checkName)
		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = kv.Keys()
		if !errors.Is(err, nats.ErrNoKeysFound) {
			t.Fatal(err.Error())
		}
	})
}
