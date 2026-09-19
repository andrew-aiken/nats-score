package checks

import (
	"fmt"
	"log/slog"

	"server/internal/logging"
	"server/internal/nats"

	natsnats "github.com/nats-io/nats.go"
)

func Remove(natsAddress string, natsCreds string, checkName string) error {
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
		slog.Debug("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	checkKey := fmt.Sprintf("check.%s", checkName)

	_, err = kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		slog.Warn("Check does not exist", "check", checkName)
		return nil
	}

	err = kv.Delete(checkKey)
	if err != nil {
		return err
	}

	fmt.Printf("Removed check %s\n", checkName)
	return nil
}
