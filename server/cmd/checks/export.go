package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"server/pkg/config"
	"server/pkg/nats"
)

func Export(directory string) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Check if directory exists if not create
	info, err := os.Stat(directory)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(directory, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !info.IsDir() {
			return fmt.Errorf("Output is not a directory")
		}
	}

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config")
		return err
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	err = natsClient.SetupConnection()
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client")
		return err
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	// List check keys
	keys, err := kv.ListKeys()
	defer keys.Stop()
	if err != nil {
		return nil
	}

	// Read keys from channel
	for key := range keys.Keys() {
		if checkName, prefix := strings.CutPrefix(key, "check."); prefix {
			log.Printf("%s", checkName)
			keyValue, err := kv.Get(key)
			if err != nil {
				return err
			}
			checkJsonBtes, err := formatCheck(keyValue.Value())

			checkPath := fmt.Sprintf("%s/%s.json", directory, checkName)

			if err := os.WriteFile(checkPath, checkJsonBtes, 0600); err != nil {
				return err
			}
		}
	}

	return err
}

func formatCheck(jsonB []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, jsonB, "", "  "); err != nil {
		return []byte{}, err
	}

	return prettyJSON.Bytes(), nil
}
