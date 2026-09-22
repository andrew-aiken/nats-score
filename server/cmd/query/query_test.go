package query_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"server/cmd/query"
	"server/internal/sink"
)

func TestList(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "results.db")

	db, err := sink.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open results database: %v", err)
	}

	rows := []sink.Row{
		{StreamSeq: 1, Subject: "results.0.noop.0", TeamID: 0, CheckName: "noop", Passed: true, Points: 5, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)},
		{StreamSeq: 2, Subject: "results.1.noop.0", TeamID: 1, CheckName: "noop", Passed: true, Points: 7, Timestamp: time.Date(2026, 1, 1, 12, 0, 5, 0, time.UTC)},
		{StreamSeq: 3, Subject: "results.1.noop.0", TeamID: 1, CheckName: "noop", Passed: true, Points: 100, Timestamp: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	if err := db.InsertBatch(context.Background(), rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close results database: %v", err)
	}

	var out bytes.Buffer
	q := sink.ScoreQuery{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	if err := query.List(&out, dbPath, q); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	want := "Team 0: 5 points\nTeam 1: 7 points\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestListNoResultsInRange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "results.db")

	var out bytes.Buffer
	q := sink.ScoreQuery{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	if err := query.List(&out, dbPath, q); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if out.String() != "" {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestMissingDatabaseFile(t *testing.T) {
	var out bytes.Buffer
	q := sink.ScoreQuery{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	err := query.List(&out, t.TempDir(), q)
	if !strings.Contains(err.Error(), "unable to open results database") {
		t.Fatalf("Expected error when opening database: %v", err)
	}
}
