package settings

import (
	"encoding/json"
	"testing"
)

func TestCheckUnmarshalJSON(t *testing.T) {
	t.Run("valid noop check", func(t *testing.T) {
		data := []byte(`{
			"name": "my-noop",
			"type": "noop",
			"description": "a noop check",
			"mutableFields": ["Pass"],
			"scoreWeight": 10,
			"definition": {"pass": false}
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Name != "my-noop" {
			t.Errorf("Name = %q, want %q", c.Name, "my-noop")
		}
		if c.ScoreWeight != 10 {
			t.Errorf("ScoreWeight = %d, want 10", c.ScoreWeight)
		}
		if c.Definition == nil {
			t.Error("Definition should not be nil")
		}
	})

	t.Run("invalidate type", func(t *testing.T) {
		data := []byte(`{
			"name": "bad-type",
			"type": 1,
			"definition": {}
		}`)

		var c Check
		err := json.Unmarshal(data, &c)
		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			t.Errorf("Failed to unmarshal incorrect type: %v", err)
		}
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "unknown-type",
			"definition": {}
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for unknown type")
		}
	})

	t.Run("missing definition returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "noop"
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for missing definition")
		}
	})

	t.Run("null definition returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "noop",
			"definition": null
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for null definition")
		}
	})

	t.Run("bad check definition", func(t *testing.T) {
		data := []byte(`{
			"name": "good-check",
			"type": "noop",
			"definition": "bad data type"
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("Expected the unmarshalling to fail due to bad check definition")
		}
	})
}
