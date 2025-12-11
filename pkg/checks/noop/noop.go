package noop

import (
	"context"

	"github.com/aaiken/nats-score/pkg/checks"
)

type Definition struct{}

type Input struct {
	Pass bool `json:"pass"`
}

func (d *Definition) Run(ctx context.Context, input map[string]interface{}) checks.Results {
	var in Input
	if err := checks.ConvertInputType(input, &in); err != nil {
		return checks.Results{
			Passed:  true,
			Message: err.Error(),
		}
	}
	return checks.Results{
		Passed:  in.Pass,
		Message: "",
	}
}
