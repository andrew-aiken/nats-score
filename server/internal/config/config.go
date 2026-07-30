package config

import (
	"encoding/json"
	"os"
)

// Config holds the application configuration
type Config struct {
	FrontendURL        string `json:"frontend_url"`
	AccountSigningSeed string `json:"account_signing_seed"`
	AccountPublicKey   string `json:"account_public_key"`
	NATSUrl            string `json:"nats_url"`
	NATSCredsFile      string `json:"nats_creds_file"`
	HttpPort           int    `json:"port"`
}

var defaultConfig = Config{
	HttpPort: 3000,
}

// Load loads the configuration from the specified JSON file
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	// var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&defaultConfig); err != nil {
		return Config{}, err
	}

	return defaultConfig, nil
}
