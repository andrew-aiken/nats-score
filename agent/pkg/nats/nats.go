package nats

import (
	"fmt"
	"log"
	"time"

	"github.com/aaiken/nats-score/pkg/config"
	"github.com/aaiken/nats-score/pkg/score"
	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	NatsUrl            string
	NatsConnectionName string
	NatsCredsFile      string
	NatsConn           *nats.Conn
	NatsKV             nats.KeyValue
	NatsKVWatcher      nats.KeyWatcher
	JetStreamConn      nats.JetStreamContext
	natsStreamSub      *nats.Subscription
}

func (n *NatsConnection) SetupConnection() error {
	err := n.natsConnect()
	if err != nil {
		return err
	}

	err = n.jetStreamConnect()
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %v", err)
	}

	if err = n.keyValueConnect(); err != nil {
		return err
	}

	return nil
}

func (n *NatsConnection) natsConnect() error {
	var nc *nats.Conn
	var err error
	var natsRetry int8 = 30

	log.Printf("Connecting to NATS at %s...", n.NatsUrl)

	// Build connection options
	opts := []nats.Option{
		nats.Name(n.NatsConnectionName),
		// Use credentials file for JWT + NKey authentication
		nats.UserCredentials(n.NatsCredsFile),
	}

	// Connect to NATS with retry
	for i := range natsRetry {
		nc, err = nats.Connect(n.NatsUrl, opts...)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to NATS (attempt %d/%d): %v", i+1, natsRetry, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to NATS after %d attempts: %v", natsRetry, err)
	}
	log.Println("Connected to NATS")

	n.NatsConn = nc

	return nil
}

func (n *NatsConnection) jetStreamConnect() error {
	js, err := n.NatsConn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %v", err)
	}

	n.JetStreamConn = js
	return nil
}

// keyValueConnect connects to JetStream settings KV and checks if the settings value is present
func (n *NatsConnection) keyValueConnect() error {
	// Get the KV bucket
	kv, err := n.JetStreamConn.KeyValue("settings")
	if err != nil {
		return fmt.Errorf("failed to get KV bucket 'settings': %v", err)
	}

	n.NatsKV = kv

	log.Println("Connected to KV bucket 'settings'")

	// Verify the "settings" key exists on startup
	if _, err := kv.Get("settings"); err != nil {
		return fmt.Errorf("required 'settings' key not found in KV bucket: %v", err)
	}
	return nil
}

// SetupKVWatcher creates a watcher for specific KVs
func (n *NatsConnection) SetupKVWatcher(keys []string) error {
	var err error

	n.NatsKVWatcher, err = n.NatsKV.WatchFiltered(keys)

	if err != nil {
		return fmt.Errorf("Failed to start KV watcher: %v", err)
	}

	return nil
}

// SubjectSubscribe subscribes to the ephemeral score trigger stream
func (n *NatsConnection) SubjectSubscribe(settings *config.Settings) error {
	var err error

	n.natsStreamSub, err = n.NatsConn.Subscribe("events.score.>", score.HandleScoreEvent(settings, n.JetStreamConn))

	if err != nil {
		return fmt.Errorf("Failed to subscribe to ephemeral events: %v", err)
	}

	log.Println("Subscribed to ephemeral events on 'events.>'")

	return nil
}

// Close closes open nats connections
func (n *NatsConnection) Close() {
	// Close KV watcher
	if n.NatsKVWatcher != nil {
		n.NatsKVWatcher.Stop()
		n.NatsKVWatcher = nil
	}

	// Close stream subscription
	if n.natsStreamSub != nil {
		n.natsStreamSub.Unsubscribe()
		n.natsStreamSub = nil
	}

	// Close nats connection
	if n.NatsConn != nil {
		n.NatsConn.Close()
		n.NatsConn = nil
	}
}
