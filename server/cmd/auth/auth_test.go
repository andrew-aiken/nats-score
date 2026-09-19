package auth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"server/cmd/auth"
	"server/internal/config"

	"github.com/nats-io/nkeys"
)

type test struct {
	name         string
	params       auth.CliParameters
	errorMessage string
}

func TestAuth(t *testing.T) {
	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatalf("failed to create account keypair: %v", err)
	}

	seed, err := kp.Seed()
	if err != nil {
		t.Fatalf("failed to get account seed: %v", err)
	}

	pubKey, err := kp.PublicKey()
	if err != nil {
		t.Fatalf("failed to get account public key: %v", err)
	}

	tests := []test{
		{
			name: "Valid",
			params: auth.CliParameters{
				ConfigFile: testGenerateConfigFile(t, config.Config{
					AccountSigningSeed: string(seed),
					AccountPublicKey:   string(pubKey),
				}),
				Standalone: false,
				Teams:      1,
			},
		},
		{
			name: "ValidStandalone",
			params: auth.CliParameters{
				ConfigFile: testGenerateConfigFile(t, config.Config{
					AccountSigningSeed: string(seed),
					AccountPublicKey:   string(pubKey),
				}),
				Standalone: true,
				Teams:      0,
			},
		},
		{
			name: "MissingConfig",
			params: auth.CliParameters{
				ConfigFile: "DNE",
				Standalone: true,
				Teams:      0,
			},
			errorMessage: "open DNE: no such file or directory",
		},
		{
			name: "InvalidSeed",
			params: auth.CliParameters{
				ConfigFile: testGenerateConfigFile(t, config.Config{
					AccountSigningSeed: "XXX",
					AccountPublicKey:   string(pubKey),
				}),
				Standalone: false,
				Teams:      0,
			},
			errorMessage: "invalid account seed: nkeys: invalid encoded key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = auth.Auth(tt.params)

			if tt.errorMessage != "" {
				if err == nil {
					t.Fatalf("Expected error message, but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Unexpected error: got '%v' want '%v'", err.Error(), tt.errorMessage)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func testGenerateConfigFile(t *testing.T, config config.Config) string {
	tmpDir := t.TempDir()

	tmpConfigFile := filepath.Join(tmpDir, "config.json")

	content, err := json.Marshal(config)
	if err != nil {
		t.FailNow()
	}

	if err := os.WriteFile(tmpConfigFile, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	return tmpConfigFile
}
