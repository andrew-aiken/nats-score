package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

// Config holds the application configuration
type Config struct {
	AccountSigningSeed string `json:"account_signing_seed"`
	AccountPublicKey   string `json:"account_public_key"`
	HttpPort           int    `json:"port"`
}

var defaultConfig = Config{
	HttpPort: 3000,
}

// Load loads the configuration from the specified JSON file
func Load(path string) (conf Config, error error) {
	file, err := os.Open(path) // #nosec G304 - fine with reading the config from anywhere on the system
	if err != nil {
		return Config{}, err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("Error closing config file", "error", err.Error())
			error = err
		}
	}()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&defaultConfig); err != nil {
		return Config{}, err
	}

	return defaultConfig, nil
}
