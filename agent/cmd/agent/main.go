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
		fmt.Println("No nats url specified using default address")
		natsURL = nats.DefaultURL
	}

	natsCredsFile := os.Getenv("NATS_CREDS_FILE")
	if natsCredsFile == "" {
		fmt.Println("No credentials file specified in env variable 'NATS_CREDS_FILE'")
		return
	}

	teamNumber := os.Getenv("TEAM_NUMBER")
	if teamNumber == "" {
		fmt.Println("No team number specified in env variable 'TEAM_NUMBER'")
		return
	}

	var agentSettings config.Settings
	var err error

	teamNumberInt, err := strconv.Atoi(teamNumber)
	if err != nil {
		log.Fatalf("Invalid TEAM_NUMBER (%v) Must be an integer", err)
	}
	agentSettings.StaticConf.TeamNumber = teamNumberInt

	log.Printf("Connecting to NATS at %s...", natsURL)

	// Build connection options
	opts := []nats.Option{
		nats.Name(teamNumber + "-agent"),
		// Use credentials file for JWT + NKey authentication
		nats.UserCredentials(natsCredsFile),
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
				// Print settings as json object
				// json.NewEncoder(os.Stdout).Encode(agentSettings)

				log.Println("Updating Global settings")

				// Remove existing checks
				// If an updated settings in kv renames or removes checks they would not be removed from the settings var
				agentSettings.Checks = map[string]config.Checks{}

				if err := json.Unmarshal(entry.Value(), &agentSettings); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
				}
			case teamNumber + ".settings":
				fmt.Println("Team-specific settings update")

				var teamSettings map[string]map[string]string
				if err := json.Unmarshal(entry.Value(), &teamSettings); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
					continue
				}

				// Replace Attributes entirely with team-specific config
				agentSettings.Attributes = teamSettings
			}
		}
	}
}
