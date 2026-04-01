package checks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"server/pkg/config"
	"server/pkg/nats"

	natsnats "github.com/nats-io/nats.go"
)

func Describe(checkName string) error {
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

	key, err := kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		return errors.New("Check does not exist")
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
