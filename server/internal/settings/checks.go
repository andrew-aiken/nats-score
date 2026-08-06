package settings

import (
	"encoding/json"
	"fmt"

	"github.com/creasty/defaults"

	"github.com/andrew-aiken/checks/helper"
)

// Check is the values for a check json object
type Check struct {
	// Parameters for the check
	Definition any `json:"definition"`
	// Additional information about the check
	Description string `json:"description"`
	// How often the check runs in seconds
	Frequency uint16 `json:"frequency"`
	// Fields in the definition that can be overwritten
	MutableFields []string `json:"mutableFields"`
	// Name of the check
	Name string `json:"name"`
	// How many points to assign the check
	ScoreWeight uint8 `json:"scoreWeight"`
	// Type of check
	Type string `json:"type"`
}

func (c *Check) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct to get the type
	type ChecksRaw struct {
		Name          string          `json:"name"`
		Definition    json.RawMessage `json:"definition"`
		Description   string          `json:"description"`
		MutableFields []string        `json:"mutableFields"`
		ScoreWeight   uint8           `json:"scoreWeight"`
		Type          string          `json:"type"`
	}

	var raw ChecksRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Name = raw.Name
	c.Description = raw.Description
	c.Type = raw.Type
	c.MutableFields = raw.MutableFields
	c.ScoreWeight = raw.ScoreWeight

	if len(raw.Definition) == 0 || string(raw.Definition) == "null" {
		return fmt.Errorf("definition not defined for check %q", raw.Name)
	}

	def, err := helper.NewDefinition(raw.Type)
	if err != nil {
		return err
	}

	// Set default values
	defaults.Set(def)

	if err := json.Unmarshal(raw.Definition, &def); err != nil {
		return fmt.Errorf("failed to unmarshal unknown definition type %q: %w", raw.Type, err)
	}
	c.Definition = def

	return nil
}
