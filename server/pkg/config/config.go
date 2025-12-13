package config

import (
	"encoding/json"
	"os"
)

// Config holds the application configuration
type Config struct {
	RedirectURL        string `json:"redirect_url"`
	FrontendURL        string `json:"frontend_url"`
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	AccountSigningSeed string `json:"account_signing_seed"`
	AccountPublicKey   string `json:"account_public_key"`
	NATSUrl            string `json:"nats_url"`
	NATSCredsFile      string `json:"nats_creds_file"`
}

// Load loads the configuration from the specified JSON file
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
