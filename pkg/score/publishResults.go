package score

import (
	"encoding/json"

	"github.com/aaiken/nats-score/pkg/checks"
	"github.com/nats-io/nats.go"
)

func publishResults(streamName string, results checks.Results, scoreWeight int8, js nats.JetStreamContext) error {
	// No points for failed check
	scoredPoints := int8(0)

	// If the check passed, award the weighter point value
	if results.Passed {
		scoredPoints = scoreWeight
	}

	data := map[string]interface{}{"timestamp": results.Timestamp, "passed": results.Passed, "points": scoredPoints, "message": results.Message, "details": results.Details}
	bytes, _ := json.Marshal(data)

	js.Publish(streamName, bytes)

	return nil
}
