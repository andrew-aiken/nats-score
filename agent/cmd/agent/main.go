package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aaiken/nats-score/pkg/config"
	"github.com/aaiken/nats-score/pkg/score"

	"github.com/nats-io/nats.go"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	natsUser := os.Getenv("NATS_USER")
	natsPass := os.Getenv("NATS_PASSWORD")
	teamNumber := os.Getenv("TEAM_NUMBER")
	if teamNumber == "" {
		teamNumber = "0"
	}

	var agentSettings config.Settings
	var err error

	teamNumberInt, err := strconv.Atoi(teamNumber)
	if err != nil {
		log.Fatalf("Invalid TEAM_NUMBER (%v) Must be an integer", err)
	}
	agentSettings.TeamNumber = teamNumberInt

	log.Printf("Connecting to NATS at %s...", natsURL)

	// Build connection options
	opts := []nats.Option{
		nats.Name(teamNumber + "-agent"),
	}
	if natsUser != "" && natsPass != "" {
		opts = append(opts, nats.UserInfo(natsUser, natsPass))
		log.Printf("Using credentials for user: %s", natsUser)
	}

	// Connect to NATS with retry
	var nc *nats.Conn
	for i := 0; i < 30; i++ {
		nc, err = nats.Connect(natsURL, opts...)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to NATS (attempt %d/30): %v", i+1, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to NATS after 30 attempts: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS")

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get the KV bucket (created by nats-setup init container)
	kv, err := js.KeyValue("settings")
	if err != nil {
		log.Fatalf("Failed to get KV bucket 'settings': %v", err)
	}
	log.Println("Connected to KV bucket 'settings'")

	// Verify the "settings" key exists on startup
	if _, err := kv.Get("settings"); err != nil {
		log.Fatalf("Required 'settings' key not found in KV bucket: %v", err)
	}

	// Watch global settings and team-specific settings
	watchList := []string{"settings", teamNumber + ".settings"}
	watcher, err := kv.WatchFiltered(watchList)

	if err != nil {
		log.Fatalf("Failed to start KV watcher: %v", err)
	}
	defer watcher.Stop()

	// Subscribe to ephemeral events (core NATS, non-JetStream)
	// These are fire-and-forget messages - only delivered if subscriber is connected
	ephemeralSub, err := nc.Subscribe("events.score.>", score.HandleScoreEvent(&agentSettings, js))
	if err != nil {
		log.Fatalf("Failed to subscribe to ephemeral events: %v", err)
	}
	defer ephemeralSub.Unsubscribe()
	log.Println("Subscribed to ephemeral events on 'events.>'")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
		ephemeralSub.Unsubscribe()
		watcher.Stop()
	}()

	// Process KV updates
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping watcher")
			return
		case entry := <-watcher.Updates():
			if entry == nil {
				// Initial sync complete
				log.Println("Initial KV sync complete, watching for updates...")
				continue
			}

			// Only process add/update operations, skip deletes and purges
			if entry.Operation() != nats.KeyValuePut {
				continue
			}

			key := entry.Key()

			switch key {
			case "settings":
				// TODO: If a key in the settings kv is removed the go struct will keep it in the config
				fmt.Println("Global settings update")

				if err := json.Unmarshal(entry.Value(), &agentSettings); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
				}

			case teamNumber + ".settings":
				fmt.Println("Team-specific settings update")

				var teamSettings struct {
					Attributes map[string]config.Attributes `json:"attributes"`
				}
				if err := json.Unmarshal(entry.Value(), &teamSettings); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
					continue
				}

				// Replace Attributes entirely with team-specific config (Checks remain untouched)
				agentSettings.Attributes = teamSettings.Attributes
				fmt.Println(agentSettings)
			}

			fmt.Println(string(entry.Value()))
		}
	}
}
