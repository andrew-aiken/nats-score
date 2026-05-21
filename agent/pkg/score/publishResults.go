package score

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/andrew-aiken/checks"
)

type PublishedResults struct {
	checks.Results
	Points uint8 `json:"points"`
}

func publishResults(streamName string, results checks.Results, scoreWeight uint8, js nats.JetStreamContext) error {
	// No points for failed check
	scoredPoints := uint8(0)
	passedSubject := 0

	// If the check passed, award the weighter point value
	if results.Passed {
		scoredPoints = scoreWeight
		passedSubject = 1
	}

	publishedResults := PublishedResults{
		Results: results,
		Points:  scoredPoints,
	}

	bytes, err := json.Marshal(publishedResults)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to marshal when attempting to publishing to %s", streamName))
		return err
	}
	_, err = js.Publish(
		fmt.Sprintf("%s.%d", streamName, passedSubject),
		bytes,
	)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed publish to stream %s", streamName))
		return err
	}

	return nil
}
