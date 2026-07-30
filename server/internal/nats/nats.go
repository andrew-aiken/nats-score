package nats

import (
	"fmt"
	"log/slog"
	"time"
	"context"

	"server/internal/settings"
	"server/internal/score"

	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	NatsUrl            string
	NatsConnectionName string
	NatsCredsFile      string
	NatsInboxPrefix    string
	NatsConn           *nats.Conn
	NatsKV             nats.KeyValue
	NatsUsersKV        nats.KeyValue
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
	var natsRetry = 30

	slog.Debug("Connecting to NATS " + n.NatsUrl)

	// Build connection options
	opts := []nats.Option{
		nats.Name(n.NatsConnectionName),
		nats.UserCredentials(n.NatsCredsFile),
	}
	if n.NatsInboxPrefix != "" {
		opts = append(opts, nats.CustomInboxPrefix(n.NatsInboxPrefix))
	}

	// Connect to NATS with retry
	for i := range natsRetry {
		nc, err = nats.Connect(n.NatsUrl, opts...)
		if err == nil {
			break
		}
		slog.Warn(fmt.Sprintf("Failed to connect to NATS (attempt %d/%d): %v", i+1, natsRetry, err))
		time.Sleep(time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to NATS after %d attempts: %v", natsRetry, err)
	}
	slog.Info("Connected to NATS")

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

	return nil
}

// SetupUsersKV connects to the "users" KV bucket. This is opt-in and must be
// called explicitly after SetupConnection() by callers that hold
// full-privilege server credentials (cmd/server, cmd/user). It is
// intentionally NOT part of SetupConnection()/keyValueConnect() because
// agent and per-team credentials (see cmd/auth/auth.go createAgentCredentials)
// are not granted any permissions on the users bucket/stream, and calling
// this from cmd/agent or cmd/checks/* would fail and break those processes.
func (n *NatsConnection) SetupUsersKV() error {
	kv, err := n.JetStreamConn.KeyValue("users")
	if err != nil {
		return fmt.Errorf("failed to get KV bucket 'users': %v", err)
	}

	n.NatsUsersKV = kv

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
func (n *NatsConnection) SubjectSubscribe(ctx context.Context, settings *settings.Settings) error {
	var err error

	var scoreStream string = "events.score.>"

	n.natsStreamSub, err = n.NatsConn.Subscribe(scoreStream, score.HandleScoreEvent(ctx, settings, n.JetStreamConn))
	if err != nil {
		return fmt.Errorf("Failed to subscribe to stream %v", err)
	}

	slog.Debug(fmt.Sprintf("Subscribed to events on '%s'", scoreStream))

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
