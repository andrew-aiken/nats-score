package checks

import (
	"log"
	"os"
	"strings"

	"server/pkg/config"
	"server/pkg/nats"
)

func List() error {
	return listChecks(false)
}

func listChecks(deleteKeys bool) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config")
		return err
	}

	// Connect to NATS settings KV
	var natsKVClient *nats.NATSKVClient
	natsKVClient, err = nats.NewNATSKVClient(cfg.NATSUrl, cfg.NATSCredsFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client")
		return err
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsKVClient.Close()
	}

	// Setup NATS key value handler
	kv := natsKVClient.GetKVClient()

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
				log.Printf("Removing key - %s", checkName)
				kv.Delete(key)
			} else {
				log.Printf("%s", checkName)
			}
		}
	}
	return err
}
