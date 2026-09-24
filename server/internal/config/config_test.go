package config_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/andrew-aiken/score/internal/config"
)

func TestLoad(t *testing.T) {
	t.Run("Expected function", func(t *testing.T) {
		appConfig, err := config.Load("./testdata/functional.json")

		if err != nil {
			t.Errorf("Failed to load data: %v", err)
		}

		if appConfig.HttpPort != 8080 {
			t.Error("Json value does not match expected result")
		}
	})

	t.Run("Incorrectly formatted config", func(t *testing.T) {
		_, err := config.Load("./testdata/error.json")

		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			t.Error("Failed to unmarshal incorrect type")
		}
	})

	t.Run("Missing config file", func(t *testing.T) {
		_, err := config.Load("./testdata/dne.json")

		if !errors.Is(err, os.ErrNotExist) {
			t.FailNow()
		}
	})
}
