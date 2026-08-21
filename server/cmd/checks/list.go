package checks

import (
	"fmt"
	"log/slog"
	"strings"

	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"
)

func List(configFile string) error {
	return listChecks(configFile, false)
}

func listChecks(configFile string, deleteKeys bool) error {
	logging.SetupLogging("warn")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	err = natsClient.SetupConnection()
	if err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	} else {
		slog.Info("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	// Generate
	keys, err := kv.ListKeys()
	defer keys.Stop()
	if err != nil {
		return nil
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {

			// Delete the check key if enabled
			if deleteKeys {
				fmt.Printf("Removing check %s\n", checkName)
				kv.Delete(key)
			} else {
				fmt.Println(checkName)
			}
		}
	}
	return err
}
