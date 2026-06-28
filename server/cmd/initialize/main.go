package initialize

import (
	"fmt"
	"log"
	"time"

	"server/pkg/config"

	"github.com/nats-io/nats.go"
)

func Initialize() error {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	opts := []nats.Option{
		nats.Name("score-server"),
	}

	// Use credentials file if provided
	if cfg.NATSCredsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.NATSCredsFile))
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSUrl, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       "settings",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		return err
	}
	log.Println("Created settings KV")

	resultsStream := nats.StreamConfig{
		Name:        "results",
		Description: "Stream of score update events",
		Subjects:    []string{"results.>"},
		MaxAge:      30 * 24 * time.Hour, // 30 days
		Replicas:    1,
		MaxMsgs:     -1,
		MaxBytes:    -1,
		MaxMsgSize:  -1,
		DenyDelete:  true,
		DenyPurge:   true,
		AllowRollup: false,
		Duplicates:  2 * time.Minute,
	}

	_, err = js.AddStream(&resultsStream)
	log.Println("Created results stream")
	if err != nil {
		return err
	}

	_, err = js.AddConsumer(resultsStream.Name, &nats.ConsumerConfig{
		Name:          "results-watcher",
		Description:   "Consumer for reading score results",
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckAllPolicy,
	})

	return err
}
