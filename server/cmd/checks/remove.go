package checks

import (
	"fmt"
	"log/slog"

	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"

	natsnats "github.com/nats-io/nats.go"
)

func Remove(checkName string) error {
	logging.SetupLogging("info")

	if checkName == "" {
		fmt.Println("Check name required")
		return nil
	}

	// Load configuration
	cfg, err := config.Load("config.json")
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
		slog.Debug("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	checkKey := fmt.Sprintf("check.%s", checkName)

	_, err = kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		slog.Warn("Check does not exist", "check", checkName)
	}

	err = kv.Delete(checkKey)
	if err != nil {
		return err
	}

	slog.Info("Removed check", "name", checkName)

	return nil
}
