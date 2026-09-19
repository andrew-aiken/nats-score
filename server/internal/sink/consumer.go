package sink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"server/internal/score"

	"github.com/nats-io/nats.go"
)

const (
	streamName   = "results"
	consumerName = "results-watcher"
	fetchBatch   = 100
	fetchWait    = 5 * time.Second
)

// Consume binds to the durable "results-watcher" pull consumer on the "results" stream and inserts every message into sqlite database
func Consume(ctx context.Context, js nats.JetStreamContext, db *DB) error {
	sub, err := js.PullSubscribe("", "", nats.Bind(streamName, consumerName))
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		if ctx.Err() != nil {
			return nil
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, fetchWait)
		msgs, err := sub.Fetch(fetchBatch, nats.Context(fetchCtx))
		cancelFetch()

		if err != nil {
			if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return err
		}

		if err := insertBatch(ctx, db, msgs); err != nil {
			return err
		}
	}
}

func insertBatch(ctx context.Context, db *DB, msgs []*nats.Msg) error {
	slog.Debug("Inserting batch of records", "length", len(msgs))

	rows := make([]Row, 0, len(msgs))
	toAck := make([]*nats.Msg, 0, len(msgs))

	for _, msg := range msgs {
		var results score.PublishedResults

		// Convert the nats objects into a golang object
		if err := json.Unmarshal(msg.Data, &results); err != nil {
			slog.Error("Failed to unmarshal published results, dropping message", "subject", msg.Subject, "error", err)
			if termErr := msg.Term(); termErr != nil {
				slog.Error("Failed to terminate, unable to parse message", "subject", msg.Subject, "error", termErr)
			}
			continue
		}

		meta, err := msg.Metadata()
		if err != nil {
			slog.Error("Failed to read message metadata, dropping message", "subject", msg.Subject, "error", err)
			if termErr := msg.Term(); termErr != nil {
				slog.Error("Failed to terminate message with missing metadata", "subject", msg.Subject, "error", termErr)
			}
			continue
		}

		teamID, checkName, err := parseSubject(msg.Subject)
		if err != nil {
			slog.Error("Failed to parse subject, dropping message", "subject", msg.Subject, "error", err)
			if termErr := msg.Term(); termErr != nil {
				slog.Error("Failed to terminate message, unable to parse subject", "subject", msg.Subject, "error", termErr)
			}
			continue
		}

		rows = append(rows, Row{
			StreamSeq: meta.Sequence.Stream,
			Subject:   msg.Subject,
			TeamID:    teamID,
			CheckName: checkName,
			Message:   results.Message,
			Passed:    results.Passed,
			Points:    results.Points,
			Details:   results.Details,
			Timestamp: results.Timestamp.Format(time.RFC3339Nano),
		})
		toAck = append(toAck, msg)
	}

	if err := db.InsertBatch(ctx, rows); err != nil {
		return err
	}

	for _, msg := range toAck {
		if err := msg.Ack(); err != nil {
			slog.Error("Failed to ack message after insert", "subject", msg.Subject, "error", err)
		}
	}

	return nil
}

// parseSubject extracts the team ID and check name from a subject
func parseSubject(subject string) (teamID uint16, checkName string, err error) {
	parts := strings.Split(subject, ".")
	if len(parts) != 4 {
		return 0, "", fmt.Errorf("expected 4 dot-separated segments, got %d", len(parts))
	}

	id, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, "", fmt.Errorf("parse team id: %w", err)
	}

	return uint16(id), parts[2], nil
}
