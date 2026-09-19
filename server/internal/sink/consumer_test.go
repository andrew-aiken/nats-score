package sink_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"server/internal/score"
	"server/internal/sink"

	"github.com/andrew-aiken/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func setupTestStream(t *testing.T) nats.JetStreamContext {
	t.Helper()

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	t.Cleanup(server.Shutdown)

	nc, err := nats.Connect(server.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	t.Cleanup(nc.Close)

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to initialize JetStream: %v", err)
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

	return js
}

func publishResult(t *testing.T, js nats.JetStreamContext, subject string, results score.PublishedResults) {
	t.Helper()

	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("Failed to marshal published results: %v", err)
	}

	if _, err := js.Publish(subject, data); err != nil {
		t.Fatalf("Failed to publish results: %v", err)
	}
}

func countResultRows(t *testing.T, dbPath string) int {
	t.Helper()

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database for verification: %v", err)
	}
	defer conn.Close()

	var count int
	if err := conn.QueryRow("SELECT count(*) FROM results").Scan(&count); err != nil {
		t.Fatalf("Failed to count result rows: %v", err)
	}

	return count
}

func waitForRowCount(t *testing.T, dbPath string, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if countResultRows(t, dbPath) >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Timed out waiting for %d result rows", want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestConsumePicksUpMessagesPublishedBeforeStart(t *testing.T) {
	js := setupTestStream(t)

	publishResult(t, js, "results.1.noop.0", score.PublishedResults{
		Results: checks.Results{Message: "passed check", Passed: true, Timestamp: time.Now()},
		Points:  5,
	})
	publishResult(t, js, "results.0.noop.1", score.PublishedResults{
		Results: checks.Results{Message: "failed check", Passed: false, Timestamp: time.Now()},
		Points:  0,
	})

	dbPath := filepath.Join(t.TempDir(), "results.db")
	db, err := sink.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() {
		done <- sink.Consume(ctx, js, db)
	}()

	waitForRowCount(t, dbPath, 2, 5*time.Second)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Consume returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Consume did not stop after context cancellation")
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database for verification: %v", err)
	}
	defer conn.Close()

	var teamID int
	var checkName string
	if err := conn.QueryRow("SELECT team_id, check_name FROM results WHERE subject = ?", "results.1.noop.0").Scan(&teamID, &checkName); err != nil {
		t.Fatalf("Failed to read parsed team_id/check_name: %v", err)
	}
	if teamID != 1 || checkName != "noop" {
		t.Fatalf("Expected team_id=1 check_name=noop, got team_id=%d check_name=%s", teamID, checkName)
	}
}

func TestConsumeSkipsUnparseableSubject(t *testing.T) {
	js := setupTestStream(t)

	// Malformed subject (missing the check-name/pass-fail segments) should be
	// terminated rather than block or crash the consumer.
	publishResult(t, js, "results.bad", score.PublishedResults{
		Results: checks.Results{Message: "malformed", Passed: true, Timestamp: time.Now()},
		Points:  1,
	})
	publishResult(t, js, "results.1.noop.0", score.PublishedResults{
		Results: checks.Results{Message: "well formed", Passed: true, Timestamp: time.Now()},
		Points:  5,
	})

	dbPath := filepath.Join(t.TempDir(), "results.db")
	db, err := sink.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() {
		done <- sink.Consume(ctx, js, db)
	}()

	waitForRowCount(t, dbPath, 1, 5*time.Second)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Consume returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Consume did not stop after context cancellation")
	}

	if got := countResultRows(t, dbPath); got != 1 {
		t.Fatalf("Expected exactly 1 row (malformed subject dropped), got %d", got)
	}
}

func TestConsumeStopsOnContextCancellation(t *testing.T) {
	js := setupTestStream(t)

	dbPath := filepath.Join(t.TempDir(), "results.db")
	db, err := sink.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- sink.Consume(ctx, js, db)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Consume returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Consume did not stop after context cancellation")
	}
}
