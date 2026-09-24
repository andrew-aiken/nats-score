package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrew-aiken/score/internal/logging"
	"github.com/andrew-aiken/score/internal/nats"
)

func Export(natsAddress string, natsCreds string, directory string) error {
	logging.SetupLogging("warn")

	// Check if directory exists if not create
	info, err := os.Stat(directory)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println(directory)
			if err := os.Mkdir(directory, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !info.IsDir() {
			return fmt.Errorf("output is not a directory")
		}
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
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

	// List check keys
	keys, err := kv.ListKeys()
	if err != nil {
		return fmt.Errorf("failed to list checks: %w", err)
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {
			fmt.Printf("Exporting check: %s\n", checkName)
			keyValue, err := kv.Get(key)
			if err != nil {
				return err
			}

			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, keyValue.Value(), "", "  "); err != nil {
				return err
			}

			fileName := fmt.Sprintf("%s.json", checkName)
			checkPath := filepath.Join(directory, fileName)

			if err := os.WriteFile(checkPath, prettyJSON.Bytes(), 0600); err != nil {
				return err
			}
		}
	}

	return err
}
