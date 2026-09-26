package agent_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/andrew-aiken/score/cmd/agent"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

type test struct {
	args         agent.RunArgs
	errorMessage string
	cmdError     string
}

func TestRun(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	address := server.Addr().String()
	nc, err := nats.Connect(address)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to initialize JetStream: %v", err)
	}

	t.Run("BadAddress", func(t *testing.T) {
		testWrapper(t, test{
			args: agent.RunArgs{
				LogLevel:    "WARN",
				NatsUrl:     "not-valid-address",
				TeamNumbers: []uint16{0, 1},
			},
			errorMessage: "Failed to connect to NATS (attempt 1/30): dial tcp: lookup not-valid-address",
		})
	})

	t.Run("MissingBucket", func(t *testing.T) {
		testWrapper(t, test{
			args: agent.RunArgs{
				LogLevel:    "ERROR",
				NatsUrl:     "nats://" + address,
				TeamNumbers: []uint16{0},
			},
			errorMessage: "Failed to connect to NATS",
			cmdError:     "failed to get KV bucket 'settings': nats: bucket not found",
		})
	})

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

	t.Run("Test", func(t *testing.T) {
		testWrapper(t, test{
			args: agent.RunArgs{
				LogLevel:    "ERROR",
				NatsUrl:     "nats://" + address,
				TeamNumbers: []uint16{0},
			},
		})
	})

	// Exercises the actual SIGINT/SIGTERM shutdown path
	// (as opposed to parent-context cancellation, which every other subtest here uses)
	t.Run("SignalShutdown", func(t *testing.T) {
		logs := captureLogs(t)

		done := make(chan error, 1)
		go func() {
			done <- agent.Run(agent.RunArgs{
				LogLevel:    "ERROR",
				NatsUrl:     "nats://" + address,
				TeamNumbers: []uint16{0},
			})
		}()

		time.Sleep(50 * time.Millisecond)

		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			t.Fatalf("Failed to send SIGINT: %v", err)
		}

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("agent.Run returned an error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("agent.Run did not return after SIGINT")
		}

		if strings.Contains(logs.String(), "NATS KV watcher failed") {
			t.Fatalf("Clean shutdown was misreported as a watcher failure:\n%s", logs.String())
		}
	})
}

func testWrapper(t *testing.T, tt test) {
	logs := captureLogs(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tt.args.Context = ctx

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := agent.Run(tt.args)
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
		t.Logf("agent.Run still running after 1s, continuing with captured logs:\n%s", logs.String())
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
