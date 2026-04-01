package checks

import (
	"fmt"
	"log"
	"os"

	"server/pkg/config"
	"server/pkg/nats"

	natsnats "github.com/nats-io/nats.go"
)

func Remove(checkName string) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if checkName == "" {
		fmt.Println("Check name required")
		return nil
	}

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

	checkKey := fmt.Sprintf("check.%s", checkName)

	_, err = kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		log.Println("Check does not exist")
	}

	err = kv.Delete(checkKey)
	if err != nil {
		return err
	}

	return nil
}
