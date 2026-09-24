package nats

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andrew-aiken/score/internal/settings"

	"github.com/nats-io/nats.go"
)

// GetChecks retrieves and parses the checks from the settings KV bucket
func GetChecks(natsKV nats.KeyValue) (map[string]settings.Check, error) {
	checks := map[string]settings.Check{}

	keys, err := natsKV.ListKeys()
	if err != nil {
		return checks, fmt.Errorf("failed to list checks: %w", err)
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {
			entry, err := natsKV.Get(key)
			if err != nil {
				return checks, fmt.Errorf("failed to get check %q: %w", checkName, err)
			}

			checkValue := settings.Check{}

			if err := json.Unmarshal(entry.Value(), &checkValue); err != nil {
				return checks, fmt.Errorf("failed to unmarshal settings: %w", err)
			}

			checks[checkName] = checkValue
		}
	}

	return checks, nil
}

// GetMutableFields returns checks that have mutableFields defined with a list of their mutable fields
func GetMutableFields(natsKV nats.KeyValue) (map[string][]string, error) {
	checks, err := GetChecks(natsKV)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for checkName, check := range checks {
		if len(check.MutableFields) > 0 {
			result[checkName] = check.MutableFields
		}
	}

	return result, nil
}

// GetTeamSettings retrieves team-specific settings from the KV bucket
func GetTeamSettings(natsKV nats.KeyValue, teamNumber string) (map[string]map[string]string, error) {
	key := fmt.Sprintf("%s.settings", teamNumber)

	settingsObj := make(map[string]map[string]string)

	entry, err := natsKV.Get(key)
	if err != nil {
		// Return empty map if key doesn't exist
		if err == nats.ErrKeyNotFound {
			return settingsObj, nil
		}
		return nil, fmt.Errorf("failed to get '%s' key: %w", key, err)
	}

	if err := json.Unmarshal(entry.Value(), &settingsObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team settings: %w", err)
	}

	return settingsObj, nil
}

// PutTeamSettings writes team-specific settings to the KV bucket
func PutTeamSettings(natsKV nats.KeyValue, teamNumber string, settingsObj map[string]map[string]string) error {
	key := fmt.Sprintf("%s.settings", teamNumber)

	data, err := json.Marshal(settingsObj)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = natsKV.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to put '%s' key: %w", key, err)
	}

	return nil
}
