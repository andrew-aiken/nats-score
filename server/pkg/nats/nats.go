package nats

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

// NATSKVClient handles NATS KV operations
type NATSKVClient struct {
	conn *nats.Conn
	js   nats.JetStreamContext
	kv   nats.KeyValue
}

// CheckConfig represents the structure of a check in the settings
type CheckConfig struct {
	// Name          string   `json:"name"`
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	ScoreWeight   int      `json:"scoreWeight"`
	MutableFields []string `json:"mutableFields"`
}

// Settings represents the structure stored in the KV bucket
type Settings struct {
	Checks map[string]CheckConfig `json:"checks"`
}

// NewNATSKVClient creates a new NATS KV client
func NewNATSKVClient(url, credsFile string) (*NATSKVClient, error) {
	opts := []nats.Option{
		nats.Name("score-server"),
	}

	// Use credentials file if provided
	if credsFile != "" {
		opts = append(opts, nats.UserCredentials(credsFile))
	}

	// Connect to NATS
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Get the KV bucket
	kv, err := js.KeyValue("settings")
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get KV bucket 'settings': %w", err)
	}

	return &NATSKVClient{
		conn: nc,
		js:   js,
		kv:   kv,
	}, nil
}

// Close closes the NATS connection
func (c *NATSKVClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// GetChecks retrieves and parses the checks from the settings KV bucket
func (c *NATSKVClient) GetChecks() (map[string]CheckConfig, error) {
	checks := map[string]CheckConfig{}

	keys, err := c.kv.ListKeys()
	defer keys.Stop()
	if err != nil {
		return checks, nil
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {
			key, err := c.kv.Get(key)
			if err != nil {
				continue
			}

			checkValue := CheckConfig{}

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
func (c *NATSKVClient) GetMutableFields() (map[string][]string, error) {
	checks, err := c.GetChecks()
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
func (c *NATSKVClient) GetTeamSettings(teamNumber string) (map[string]map[string]string, error) {
	key := fmt.Sprintf("%s.settings", teamNumber)

	entry, err := c.kv.Get(key)
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
func (c *NATSKVClient) PutTeamSettings(teamNumber string, settings map[string]map[string]string) error {
	key := fmt.Sprintf("%s.settings", teamNumber)

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = c.kv.Put(key, data)
	if err != nil {
		return fmt.Errorf("failed to put '%s' key: %w", key, err)
	}

	return nil
}

func (c *NATSKVClient) GetKVClient() nats.KeyValue {
	return c.kv
}

func (c *NATSKVClient) GetNATSClient() *nats.Conn {
	return c.conn
}
