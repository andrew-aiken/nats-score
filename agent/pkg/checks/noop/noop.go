package noop

import (
	"context"
	"time"

	"github.com/aaiken/nats-score/pkg/checks"
)

type Definition struct{}

type Input struct {
	Pass bool `json:"pass"`
}

func (d *Definition) Run(ctx context.Context, input map[string]interface{}) checks.Results {
	result := checks.Results{
		Timestamp: time.Now(),
		Passed:    true,
	}

	var userInput Input
	userInput.Pass = true

	if err := checks.ConvertInputType(input, &userInput); err != nil {
		result.Message = err.Error()
		result.Passed = false
		return result
	}

	result.Passed = userInput.Pass

	return result
}
