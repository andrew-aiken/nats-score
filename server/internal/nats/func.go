package nats

import (
	"encoding/json"
	"fmt"
	"strings"

	"server/internal/settings"

	"github.com/nats-io/nats.go"
)

// GetChecks retrieves and parses the checks from the settings KV bucket
func GetChecks(natsKV nats.KeyValue) (map[string]settings.Check, error) {
	checks := map[string]settings.Check{}

	keys, err := natsKV.ListKeys()
	defer keys.Stop()
	if err != nil {
		return checks, nil
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {
			key, err := natsKV.Get(key)
			if err != nil {
				continue
			}

			checkValue := settings.Check{}

			if err := json.Unmarshal(key.Value(), &checkValue); err != nil {
				return checks, fmt.Errorf("failed to unmarshal settings: %w", err)
			}

			checks[checkName] = checkValue
		}
	}

	return checks, nil
}

// GetMutableFields returns a map of check names to their mutable fields
// Only includes checks that have mutableFields defined
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
// The key format is "{teamNumber}.settings"
func GetTeamSettings(natsKV nats.KeyValue, teamNumber string) (map[string]map[string]string, error) {
	key := fmt.Sprintf("%s.settings", teamNumber)

	entry, err := natsKV.Get(key)
	if err != nil {
		// Return empty map if key doesn't exist
		if err == nats.ErrKeyNotFound {
			return make(map[string]map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get '%s' key: %w", key, err)
	}

	var settings map[string]map[string]string
	if err := json.Unmarshal(entry.Value(), &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal team settings: %w", err)
	}

	return settings, nil
}

// PutTeamSettings writes team-specific settings to the KV bucket
// The key format is "{teamNumber}.settings"
func PutTeamSettings(natsKV nats.KeyValue, teamNumber string, settings map[string]map[string]string) error {
	key := fmt.Sprintf("%s.settings", teamNumber)

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = natsKV.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to put '%s' key: %w", key, err)
	}

	return nil
}
