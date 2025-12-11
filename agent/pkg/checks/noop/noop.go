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

	var in Input
	if err := checks.ConvertInputType(input, &in); err != nil {
		result.Message = err.Error()
		return result
	}
	result.Passed = in.Pass

	return result
}
