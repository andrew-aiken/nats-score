package checks_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"server/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestExport(t *testing.T) {
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

	checkExportDir := t.TempDir()

	t.Run("NoNats", func(t *testing.T) {
		err := checks.Export("DNE", "", checkExportDir)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "failed to connect to NATS after 3 attempts: dial tcp: lookup DNE: no such host") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoDir", func(t *testing.T) {
		dneDir := filepath.Join(checkExportDir, "dne", "dne")
		err := checks.Export(natsServerAddress, "", dneDir)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), fmt.Sprintf("mkdir %s: no such file or directory", dneDir)) {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("DirIsFile", func(t *testing.T) {
		checkDirFile := filepath.Join(checkExportDir, "file")

		_, err := os.Create(checkDirFile)
		if err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		err = checks.Export(natsServerAddress, "", checkDirFile)

		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "output is not a directory") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("BadJson", func(t *testing.T) {
		_, err = kv.Put("check.noop", []byte("{]"))
		if err != nil {
			t.Fatal(err.Error())
		}

		err = checks.Export(natsServerAddress, "", checkExportDir)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "invalid character ']' looking for beginning of object key string") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("BadFileWrite", func(t *testing.T) {
		_, err = kv.Put("check.noop", []byte("{}"))
		if err != nil {
			t.Fatal(err.Error())
		}

		noopCheckDirFile := filepath.Join(checkExportDir, "noop.json")
		err = os.Mkdir(noopCheckDirFile, 0666)
		if err != nil {
			t.Fatal(err.Error())
		}

		err = checks.Export(natsServerAddress, "", checkExportDir)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("open %s: is a directory", noopCheckDirFile)) {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})
}
