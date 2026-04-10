package score

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/aaiken/nats-score/checks"
)

type PublishedResults struct {
	checks.Results
	Points int8 `json:"points"`
}

func publishResults(streamName string, results checks.Results, scoreWeight int8, js nats.JetStreamContext) error {
	// No points for failed check
	scoredPoints := int8(0)
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

	bytes, _ := json.Marshal(publishedResults)
	js.Publish(
		fmt.Sprintf("%s.%d", streamName, passedSubject),
		bytes,
	)

	return nil
}
