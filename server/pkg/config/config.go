package config

import (
	"encoding/json"
	"os"
)

// Config holds the application configuration
type Config struct {
	RedirectURL        string         `json:"redirect_url"`
	FrontendURL        string         `json:"frontend_url"`
	ClientID           string         `json:"client_id"`
	ClientSecret       string         `json:"client_secret"`
	AccountSigningSeed string         `json:"account_signing_seed"`
	AccountPublicKey   string         `json:"account_public_key"`
	NATSUrl            string         `json:"nats_url"`
	NATSCredsFile      string         `json:"nats_creds_file"`
	DiscordGuildID     string         `json:"discord_guild_id"`
	DiscordRoleMap     DiscordRoleMap `json:"discord_role_map"`
	StaticAuthMap      StaticAuthMap  `json:"static_auth"`
	HttpPort           int            `json:"port"`
}

var defaultConfig = Config{
	HttpPort: 3000,
}

type DiscordRoleMap map[string]string

type StaticAuthMap map[string]string

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
