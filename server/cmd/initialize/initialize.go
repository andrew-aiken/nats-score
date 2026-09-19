package initialize

import (
	"fmt"
	"log/slog"
	"time"

	"server/cmd/user"
	"server/internal/config"
	"server/internal/logging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func Initialize(configPath string) error {
	logging.SetupLogging("info")

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return err
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
	slog.Info("Successfully created NATS settings KV")

	_, err = js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:       "users",
		Description:  "Username/password login account storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		return err
	}
	slog.Info("Successfully created NATS users KV")

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
	if err != nil {
		return err
	}
	slog.Info("Successfully created results NATS stream")

	_, err = js.AddConsumer(resultsStream.Name, &nats.ConsumerConfig{
		Name:          "results-watcher",
		Description:   "Consumer for reading score results",
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		MaxAckPending: 500,
	})
	if err != nil {
		return err
	}
	slog.Debug("Successfully created consumer")

	// Generate random uuid what will server as the admin password
	password := fmt.Sprint(uuid.New())

	// Add admin user
	err = user.Add(configPath, "admin", "admin", password, true)
	if err != nil {
		return err
	}

	slog.Info("Generated initial admin user credentials", "password", password)

	return err
}
