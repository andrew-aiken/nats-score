package server_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"server/cmd/server"
	"server/internal/config"
	"server/internal/score"

	"github.com/andrew-aiken/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

// TestServerWithResultsDB exercises the "score server start --db" path
// end-to-end: it starts the real server.Server(), lets it bind to the
// results-watcher consumer, publishes a score result, and confirms the row
// lands in the SQLite database before verifying a clean shutdown.
func TestServerWithResultsDB(t *testing.T) {
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

	// Mirror cmd/initialize/initialize.go's setup
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: "settings"}); err != nil {
		t.Fatalf("Failed to create settings KV: %v", err)
	}
	if _, err := js.CreateKeyValue(&nats.KeyValueConfig{Bucket: "users"}); err != nil {
		t.Fatalf("Failed to create users KV: %v", err)
	}
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     "results",
		Subjects: []string{"results.>"},
	}); err != nil {
		t.Fatalf("Failed to create results stream: %v", err)
	}
	if _, err := js.AddConsumer("results", &nats.ConsumerConfig{
		Name:          "results-watcher",
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		MaxAckPending: 500,
	}); err != nil {
		t.Fatalf("Failed to create results-watcher consumer: %v", err)
	}

	testDir := t.TempDir()

	cfg := config.Config{
		HttpPort:           1338,
		AccountSigningSeed: "SAAFFOSIG6JRRWW3N3OX54TQBYCUAZAI4LAX2OXBCOO52PXM3CGLPSMFAM",
		AccountPublicKey:   "ACTQ6KLZTMWN46EM6QVXBBGE45UTAKJIZUXYB3ULTSFLMMM2C63MPNWO",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configFile := filepath.Join(testDir, "config.json")
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	dbPath := filepath.Join(testDir, "results.db")

	ctx, cancel := context.WithCancel(context.Background())

	args := server.ServerArgs{
		NatsAddress:    address,
		LogLevel:       "DEBUG",
		ConfigFilePath: configFile,
		Context:        ctx,
		DB:             true,
		DBPath:         dbPath,
	}

	done := make(chan error, 1)
	go func() { done <- server.Server(args) }()

	// Publish a result once the results stream has at least a chance to be bound;
	// DeliverAllPolicy means order relative to subscribe doesn't matter.
	published, err := json.Marshal(score.PublishedResults{
		Results: checks.Results{Message: "integration test", Passed: true, Timestamp: time.Now()},
		Points:  3,
	})
	if err != nil {
		t.Fatalf("Failed to marshal published results: %v", err)
	}
	if _, err := js.Publish("results.1.noop.0", published); err != nil {
		t.Fatalf("Failed to publish results: %v", err)
	}

	waitForResultRow(t, dbPath, 3*time.Second)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server.Server returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server.Server did not shut down after context cancellation")
	}
}

func waitForResultRow(t *testing.T, dbPath string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(dbPath); err == nil {
			conn, err := sql.Open("sqlite", dbPath)
			if err == nil {
				var count int
				scanErr := conn.QueryRow("SELECT count(*) FROM results").Scan(&count)
				conn.Close()
				if scanErr == nil && count >= 1 {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("Timed out waiting for a result row to be written")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
