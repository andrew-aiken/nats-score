package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"server/cmd/server"
	"server/internal/config"
	"server/internal/settings"

	"github.com/andrew-aiken/checks/noop"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

type test struct {
	config       config.Config
	natsAddress  string
	errorMessage string
	cmdError     string
}

func TestServer(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	natsServer := natsserver.RunServer(&opts)
	defer natsServer.Shutdown()

	address := natsServer.Addr().String()
	nc, err := nats.Connect(address)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to initialize JetStream: %v", err)
	}

	t.Run("InvalidConfig", func(t *testing.T) {
		err := server.Server(server.ServerArgs{
			LogLevel:       "WARN",
			ConfigFilePath: "DNE",
		})
		if err != nil {
			if !strings.Contains(err.Error(), "load config: open DNE: no such file or directory") {
				t.Error(err.Error())
			}
		}
	})

	t.Run("MissingKeys", func(t *testing.T) {
		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort: 1337,
			},
			errorMessage: "Failed to initialize NATS auth service: invalid account seed: nkeys: invalid encoded key",
		})
	})

	t.Run("BadAddress", func(t *testing.T) {
		testWrapper(t, test{
			natsAddress: "not-valid-address",
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
			errorMessage: "Failed to connect to NATS (attempt 1/30): dial tcp: lookup not-valid-address",
		})
	})

	t.Run("MissingSettingsBucket", func(t *testing.T) {
		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
			errorMessage: "Failed to initialize NATS KV client",
			cmdError:     "failed to get KV bucket 'settings': nats: bucket not found",
		})
	})

	settingsBucket, err := js.CreateKeyValue(&nats.KeyValueConfig{
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

	t.Run("MissingUsersBucket", func(t *testing.T) {
		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
			errorMessage: "Failed to initialize NATS users KV client",
			cmdError:     "failed to get KV bucket 'users': nats: bucket not found",
		})
	})

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       "users",
		Description:  "Username/password login account storage",
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

	t.Run("CheckAdd", func(t *testing.T) {
		check := settings.Check{
			Name:       "Noop",
			Type:       "noop",
			Frequency:  30,
			Definition: noop.Definition{},
		}

		checkBytes, err := json.Marshal(check)
		if err != nil {
			t.Fatal(err)
		}

		_, err = kv.Put("check.noop", checkBytes)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(5 * time.Millisecond)

		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
		})
	})

	time.Sleep(5 * time.Millisecond)

	t.Run("CheckDelete", func(t *testing.T) {
		err = kv.Delete("check.noop")
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(5 * time.Millisecond)

		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
		})
	})

	t.Run("BadCheckJson", func(t *testing.T) {
		checkBytes, err := json.Marshal("{]")
		if err != nil {
			t.Fatal(err)
		}

		_, err = kv.Put("check.bad", checkBytes)
		if err != nil {
			t.Fatal(err)
		}

		testWrapper(t, test{
			natsAddress: address,
			config: config.Config{
				HttpPort:           1337,
				AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
				AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
			},
			errorMessage: "Failed to unmarshal settings",
		})
	})

	time.Sleep(time.Second)
}

func testWrapper(t *testing.T, tt test) {
	logs := captureLogs(t)

	testDir := t.TempDir()

	data, err := json.Marshal(tt.config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %s", err.Error())
	}

	configFile := filepath.Join(testDir, "config.json")
	err = os.WriteFile(configFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := server.ServerArgs{
		LogLevel:       "DEBUG",
		ConfigFilePath: configFile,
		NatsAddress:    tt.natsAddress,
		Context:        ctx,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := server.Server(args)
		if err != nil {
			if !strings.Contains(err.Error(), tt.cmdError) {
				t.Errorf("Unexpected command error: %s\nLogs:\n%s", err.Error(), logs.String())
			}
		}
	}()

	got := waitForLog(logs, tt.errorMessage, time.Second)
	if tt.errorMessage != "" && !strings.Contains(got, tt.errorMessage) {
		t.Errorf("Expected error log not found:\nLogs:\n%s", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Logf("server.Server still running after 1s, continuing with captured logs:\n%s", logs.String())
	}
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func captureLogs(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %s", err.Error())
	}

	origStdout := os.Stdout
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := io.Copy(buf, r)
		if err != nil {
			t.Error(err.Error())
		}
	}()

	t.Cleanup(func() {
		os.Stdout = origStdout
		err := w.Close()
		if err != nil {
			t.Error(err.Error())
		}
		<-done
		err = r.Close()
		if err != nil {
			t.Error(err.Error())
		}
	})

	return buf
}

func waitForLog(buf *syncBuffer, substr string, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for {
		got := buf.String()
		if strings.Contains(got, substr) || time.Now().After(deadline) {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
}
