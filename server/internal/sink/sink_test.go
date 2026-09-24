package sink_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/andrew-aiken/score/internal/sink"
)

func openTestDB(t *testing.T) *sink.DB {
	t.Helper()

	db, err := sink.Open(filepath.Join(t.TempDir(), "results.db"))
	if err != nil {
		t.Fatalf("Failed to open results database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestInsertBatch(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	rows := []sink.Row{
		{
			StreamSeq: 1,
			Subject:   "results.0.noop.0",
			TeamID:    0,
			CheckName: "noop",
			Message:   "dummy",
			Passed:    true,
			Points:    5,
			Details:   map[string]string{"key": "value"},
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			StreamSeq: 2,
			Subject:   "results.1.noop.1",
			TeamID:    1,
			CheckName: "noop",
			Message:   "failed",
			Passed:    false,
			Points:    0,
			Details:   nil,
			Timestamp: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		},
	}

	if err := db.InsertBatch(ctx, rows); err != nil {
		t.Fatalf("Failed to insert batch: %v", err)
	}
}

func TestInsertBatchIgnoresDuplicateStreamSeq(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	row := sink.Row{
		StreamSeq: 1,
		Subject:   "results.0.noop.0",
		TeamID:    0,
		CheckName: "noop",
		Message:   "dummy",
		Passed:    true,
		Points:    5,
		Details:   map[string]string{"key": "value"},
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := db.InsertBatch(ctx, []sink.Row{row}); err != nil {
		t.Fatalf("Failed to insert row: %v", err)
	}

	// Redelivery of the same stream sequence should be a no-op, not an error.
	row.Message = "changed on redelivery"
	if err := db.InsertBatch(ctx, []sink.Row{row}); err != nil {
		t.Fatalf("Failed to insert duplicate row: %v", err)
	}
}

func TestInsertBatchEmpty(t *testing.T) {
	db := openTestDB(t)

	if err := db.InsertBatch(context.Background(), nil); err != nil {
		t.Fatalf("Failed to insert empty batch: %v", err)
	}
}

func TestTeamScores(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	rows := []sink.Row{
		// Half a second after the query's start boundary: must still be included, exercising the datetime() normalization fix for RFC3339Nano's fractional-second suffix.
		{StreamSeq: 1, Subject: "results.0.noop.0", TeamID: 0, CheckName: "noop", Passed: true, Points: 5, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 500_000_000, time.UTC)},
		{StreamSeq: 2, Subject: "results.0.ping.0", TeamID: 0, CheckName: "ping", Passed: true, Points: 3, Timestamp: time.Date(2026, 1, 1, 12, 0, 10, 0, time.UTC)},
		{StreamSeq: 3, Subject: "results.1.noop.0", TeamID: 1, CheckName: "noop", Passed: true, Points: 7, Timestamp: time.Date(2026, 1, 1, 12, 0, 5, 0, time.UTC)},
		// Outside the query range used below.
		{StreamSeq: 4, Subject: "results.1.noop.0", TeamID: 1, CheckName: "noop", Passed: true, Points: 100, Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	if err := db.InsertBatch(ctx, rows); err != nil {
		t.Fatalf("Failed to insert rows: %v", err)
	}

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)

	t.Run("range includes fractional-second boundary row and excludes later rows", func(t *testing.T) {
		scores, err := db.TeamScores(ctx, sink.ScoreQuery{Start: start, End: end})
		if err != nil {
			t.Fatalf("TeamScores failed: %v", err)
		}
		want := []sink.TeamScore{{TeamID: 0, Points: 8}, {TeamID: 1, Points: 7}}
		if !reflect.DeepEqual(scores, want) {
			t.Fatalf("got %+v, want %+v", scores, want)
		}
	})

	t.Run("team filter", func(t *testing.T) {
		team := uint16(0)
		scores, err := db.TeamScores(ctx, sink.ScoreQuery{Start: start, End: end, TeamID: &team})
		if err != nil {
			t.Fatalf("TeamScores failed: %v", err)
		}
		want := []sink.TeamScore{{TeamID: 0, Points: 8}}
		if !reflect.DeepEqual(scores, want) {
			t.Fatalf("got %+v, want %+v", scores, want)
		}
	})

	t.Run("check filter", func(t *testing.T) {
		scores, err := db.TeamScores(ctx, sink.ScoreQuery{Start: start, End: end, CheckName: "ping"})
		if err != nil {
			t.Fatalf("TeamScores failed: %v", err)
		}
		want := []sink.TeamScore{{TeamID: 0, Points: 3}}
		if !reflect.DeepEqual(scores, want) {
			t.Fatalf("got %+v, want %+v", scores, want)
		}
	})

	t.Run("empty range", func(t *testing.T) {
		scores, err := db.TeamScores(ctx, sink.ScoreQuery{
			Start: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("TeamScores failed: %v", err)
		}
		if len(scores) != 0 {
			t.Fatalf("expected no scores, got %+v", scores)
		}
	})
}
