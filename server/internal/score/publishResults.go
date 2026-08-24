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

const (
	ScorePassed uint8 = iota
	ScoreFailed
)

func publishResults(streamName string, results checks.Results, scoreWeight uint8, js nats.JetStreamContext) error {
	// No points for failed check
	var scoredPoints uint8 = 0
	passedSubject := ScoreFailed

	// If the check passed, award the weighter point value
	if results.Passed {
		scoredPoints = scoreWeight
		passedSubject = ScorePassed
	}

	publishedResults := PublishedResults{
		Results: results,
		Points:  scoredPoints,
	}

	bytes, err := json.Marshal(publishedResults)
	if err != nil {
		slog.Error("Failed to marshal results object")
		return err
	}

	scoreSubject := fmt.Sprintf("%s.%d", streamName, passedSubject)
	_, err = js.Publish(scoreSubject, bytes)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed publish to results stream %s", streamName))
		return err
	}

	return nil
}
