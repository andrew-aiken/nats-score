package config

import (
	"encoding/json"
	"fmt"

	"github.com/creasty/defaults"

	"github.com/aaiken/nats-score/pkg/checks/dns"
	"github.com/aaiken/nats-score/pkg/checks/http"
	"github.com/aaiken/nats-score/pkg/checks/icmp"
	"github.com/aaiken/nats-score/pkg/checks/noop"
)

type Settings struct {
	Checks     map[string]Checks     `json:"checks"`
	Attributes map[string]Attributes `json:"attributes"`
	TeamNumber int
}

type Attributes struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type Checks struct {
	Description string      `json:"description"`
	Type        string      `json:"type"`
	ScoreWeight int8        `json:"score_weight"`
	Definition  interface{} `json:"definition"`
}

func (c *Checks) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct to get the type
	type ChecksRaw struct {
		Description string          `json:"description"`
		Type        string          `json:"type"`
		ScoreWeight int8            `json:"score_weight"`
		Definition  json.RawMessage `json:"definition"`
	}

	var raw ChecksRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Description = raw.Description
	c.Type = raw.Type
	c.ScoreWeight = raw.ScoreWeight

	var def interface{}

	// Skip unmarshalling if definition is empty or null
	if len(raw.Definition) == 0 || string(raw.Definition) == "null" {
		c.Definition = &noop.Definition{}
		return nil
	}

	// type Check interface{}

	// Unmarshal Definition based on Type
	switch raw.Type {
	case "dns":
		def = &dns.Definition{}
	case "http":
		def = &http.Definition{}
	case "icmp":
		def = &icmp.Definition{}
	case "noop":
		def = &noop.Definition{}
	default:
		// For unknown types, keep as raw JSON map
		def = &noop.Definition{}
	}

	// Set default values
	defaults.Set(def)

	if err := json.Unmarshal(raw.Definition, &def); err != nil {
		return fmt.Errorf("failed to unmarshal unknown definition type %q: %w", raw.Type, err)
	}
	c.Definition = def

	return nil
}
