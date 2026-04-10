package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/creasty/defaults"
	"github.com/nats-io/nats.go"

	"github.com/aaiken/nats-score/checks/dns"
	"github.com/aaiken/nats-score/checks/http"
	"github.com/aaiken/nats-score/checks/icmp"
	"github.com/aaiken/nats-score/checks/noop"
	"github.com/aaiken/nats-score/checks/ssh"
	"github.com/aaiken/nats-score/checks"
)

type Settings struct {
	Checks     map[string]Check             `json:"checks"`
	Attributes map[string]map[string]string `json:"attributes"`
	StaticConf checks.StaticConf          `json:"static_conf"`
}

type Check struct {
	Name          string   `json:"name"`
	Definition    any      `json:"definition"`
	Description   string   `json:"description"`
	MutableFields []string `json:"mutableFields"`
	ScoreWeight   int8     `json:"scoreWeight"`
	Type          string   `json:"type"`
}

func (c *Check) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct to get the type
	type ChecksRaw struct {
		Name          string          `json:"name"`
		Description   string          `json:"description"`
		Type          string          `json:"type"`
		MutableFields []string        `json:"mutableFields"`
		ScoreWeight   int8            `json:"scoreWeight"`
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

	if s.Checks == nil {
		s.Checks = make(map[string]Check)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context cancelled, stopping watcher")
			return
		case entry := <-natsKVWatcher.Updates():
			if entry == nil {
				// Initial sync complete
				slog.Info("Initial KV sync complete, watching for updates...")
				continue
			}

			// Only process add/update operations, skip deletes and purges
			if entry.Operation() != nats.KeyValuePut {
				continue
			}

			key := entry.Key()
			checkName, isCheck := strings.CutPrefix(key, "check.")
			value := entry.Value()
			switch {
			case key == teamSettingKey:
				slog.Info("Team settings update")

				var teamSettings map[string]map[string]string
				if err := json.Unmarshal(entry.Value(), &teamSettings); err != nil {
					slog.Warn("Failed to unmarshal settings for setting %s: %v", key, err)
					continue
				}

				// Replace Attributes entirely with team-specific config
				s.Attributes = teamSettings
			case isCheck:
				slog.Info(fmt.Sprintf("Updating check %s", checkName))

				check := Check{}

				if err := json.Unmarshal(value, &check); err != nil {
					slog.Warn("Failed to unmarshal settings for check %s: %v", checkName, err)
				}

				s.Checks[checkName] = check
			}
		}
	}
}
