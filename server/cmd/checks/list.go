package checks

import (
	"fmt"
	"log/slog"
	"strings"

	"server/internal/logging"
	"server/internal/nats"
)

func List(natsAddress string, natsCreds string) error {
	return listChecks(natsAddress, natsCreds, false)
}

func listChecks(natsAddress string, natsCreds string, deleteKeys bool) error {
	logging.SetupLogging("warn")

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
	}
	err := natsClient.SetupConnection()
	if err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	} else {
		slog.Info("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	keys, err := kv.ListKeys()
	if err != nil {
		return fmt.Errorf("failed to list checks: %w", err)
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {

			// Delete the check key if enabled
			if deleteKeys {
				fmt.Printf("Removing check %s\n", checkName)
				err = kv.Delete(key)
				if err != nil {
					slog.Error("Failed to delete checks nats key")
					return err
				}
			} else {
				fmt.Println(checkName)
			}
		}
	}
	return nil
}
