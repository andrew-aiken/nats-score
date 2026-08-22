package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"

	natsnats "github.com/nats-io/nats.go"
)

func Describe(configFile string, checkName string) error {
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
		slog.Debug("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	checkKey := fmt.Sprintf("check.%s", checkName)

	key, err := kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		return errors.New("check does not exist")
	} else if err != nil {
		return err
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, key.Value(), "", "  "); err != nil {
		return err
	}
	fmt.Println(prettyJSON.String())

	return nil
}
