package checks_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrew-aiken/score/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestImport(t *testing.T) {
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

	t.Run("Missing Settings Bucket", func(t *testing.T) {
		err := checks.Import(natsServerAddress, "", "./testdata/")

		if !strings.Contains(err.Error(), "failed to get KV bucket 'settings'") {
			t.Fatalf("Got wrong error message: %v", err.Error())
		}
	})

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
		err := checks.Import("DNE", "", t.TempDir())

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "failed to connect to NATS after 3 attempts: dial tcp: lookup DNE") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoDir", func(t *testing.T) {
		dneDir := filepath.Join(t.TempDir(), "dne")
		err := checks.Import(natsServerAddress, "", dneDir)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), fmt.Sprintf("stat %s: no such file or directory", dneDir)) {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("DirIsFile", func(t *testing.T) {
		checkDirFile := filepath.Join(t.TempDir(), "file")

		_, err := os.Create(checkDirFile)
		if err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		err = checks.Import(natsServerAddress, "", checkDirFile)

		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("open %s: not a directory", checkDirFile)) {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("BadJson", func(t *testing.T) {
		checkDir := t.TempDir()
		checkFile := filepath.Join(checkDir, "noop.json")
		checkData := "{]"

		err := os.WriteFile(checkFile, []byte(checkData), 0666)
		if err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		err = checks.Import(natsServerAddress, "", checkDir)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "invalid character ']' looking for beginning of object key string") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("DirCheck", func(t *testing.T) {
		checkDir := t.TempDir()
		checkDirFile := filepath.Join(checkDir, "noop.json")

		err := os.Mkdir(checkDirFile, 0666)
		if err != nil {
			t.Fatalf("Error creating file: %s", err.Error())
		}

		err = checks.Import(natsServerAddress, "", checkDir)

		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("read %s: is a directory", checkDirFile)) {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})
}
