package checks

import (
	"context"
	"encoding/json"
	"time"
)

// Checker is the interface that all check types must implement
type Checker interface {
	Run(ctx context.Context, input map[string]interface{}) Results
}

type Results struct {
	Details   map[string]string
	Message   string
	Passed    bool
	Timestamp time.Time
}

func ConvertInputType(input map[string]interface{}, output interface{}) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, output); err != nil {
		return err
	}
	return nil
}
