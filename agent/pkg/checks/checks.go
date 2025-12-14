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
	Details   map[string]string `json:"details"`
	Message   string            `json:"message"`
	Passed    bool              `json:"passed"`
	Timestamp time.Time         `json:"timestamp"`
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
