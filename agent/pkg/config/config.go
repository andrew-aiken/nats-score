package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/creasty/defaults"
	"github.com/nats-io/nats.go"

	"github.com/aaiken/nats-score/pkg/checks/dns"
	"github.com/aaiken/nats-score/pkg/checks/http"
	"github.com/aaiken/nats-score/pkg/checks/icmp"
	"github.com/aaiken/nats-score/pkg/checks/noop"
	"github.com/aaiken/nats-score/pkg/checks/ssh"
	"github.com/aaiken/nats-score/pkg/settings"
)

type Settings struct {
	Checks     map[string]Checks            `json:"checks"`
	Attributes map[string]map[string]string `json:"attributes"`
	StaticConf settings.StaticConf          `json:"static_conf"`
}

type Checks struct {
	Name          string   `json:"name"`
	Definition    any      `json:"definition"`
	Description   string   `json:"description"`
	MutableFields []string `json:"mutable_fields"`
	ScoreWeight   int8     `json:"score_weight"`
	Type          string   `json:"type"`
}

func (c *Checks) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct to get the type
	type ChecksRaw struct {
		Name          string          `json:"name"`
		Description   string          `json:"description"`
		Type          string          `json:"type"`
		MutableFields []string        `json:"mutable_fields"`
		ScoreWeight   int8            `json:"score_weight"`
		Definition    json.RawMessage `json:"definition"`
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

	var def any

	// Skip unmarshalling if definition is empty or null
	if len(raw.Definition) == 0 || string(raw.Definition) == "null" {
		c.Definition = &noop.Definition{}
		return nil
	}

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
	case "ssh":
		def = &ssh.Definition{}
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

// MonitorSettings loops monitoring the nats KV for settings updates
func (s *Settings) MonitorSettings(ctx context.Context, teamNumber string, natsKVWatcher nats.KeyWatcher) {
	var teamSettingKey string = teamNumber + ".settings"

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping watcher")
			return
		case entry := <-natsKVWatcher.Updates():
			if entry == nil {
				// Initial sync complete
				log.Println("Initial KV sync complete, watching for updates...")
				continue
			}

			// Only process add/update operations, skip deletes and purges
			if entry.Operation() != nats.KeyValuePut {
				continue
			}

			key := entry.Key()

			switch key {
			case "settings":
				// Print settings as json object
				// json.NewEncoder(os.Stdout).Encode(s)

				log.Println("Updating Global settings")

				// This removes existing checks
				// If an updated settings in kv renames or removes checks they would not be removed from the settings var
				// In theory their could be a race condition here, but its a pretty low risk
				s.Checks = map[string]Checks{}

				if err := json.Unmarshal(entry.Value(), &s); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
				}
			case teamSettingKey:
				log.Println("Team-specific settings update")

				var teamSettings map[string]map[string]string
				if err := json.Unmarshal(entry.Value(), &teamSettings); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
					continue
				}

				// Replace Attributes entirely with team-specific config
				s.Attributes = teamSettings
			}
		}
	}
}
