package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"server/pkg/config"
	"server/pkg/cron"

	"server/pkg/handlers"
	"server/pkg/middleware"
	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
	"golang.org/x/oauth2"
)

type Check struct {
	Frequency   int16  `json:"frequency"`   // How often the check runs in seconds
	Type        string `json:"type"`        // Type of check
	Description string `json:"description"` // Additional information about the check
	ScoreWeight int8   `json:"scoreWeight"` // How many points to assign the check
}

// TODO
// This is the state key used for security, sent in login, validated in callback.
// For this example we keep it simple and hardcode a string
// but in real apps you must provide a proper function that generates a state.
const state = "random"

func Server() error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		fmt.Printf("Failed to create cron scheduler: %v\n", err)
	}

	defer cronScheduler.Shutdown()

	// Initialize NATS auth service
	natsAuthService, err := nats.NewNATSAuthService(cfg.AccountSigningSeed, cfg.AccountPublicKey)
	if err != nil {
		log.Fatalf("Failed to initialize NATS auth service: %v", err)
	}

	// Initialize NATS KV client (optional - only if NATS URL is configured)
	// TODO: Change this logic
	var natsKVClient *nats.NATSKVClient
	natsKVClient, err = nats.NewNATSKVClient(cfg.NATSUrl, cfg.NATSCredsFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client\nMutable fields endpoint will be unavailable")
		return err
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsKVClient.Close()
	}

	kv := natsKVClient.GetKVClient()
	watcher, err := kv.Watch("check.*")
	if err != nil {
		log.Fatalf("Failed to start KV watcher: %v", err)
	}
	defer watcher.Stop()

	// Watch for check updates in background
	go func() {
		initialized := false
		for entry := range watcher.Updates() {
			if entry == nil {
				initialized = true
				continue
			}

			// Remove the prefix from the nats KV
			checkName := strings.TrimPrefix(entry.Key(), "check.")

			// Filter based on event type
			switch entry.Operation().String() {
			case "KeyValuePutOp":
				var check Check

				if err := json.Unmarshal(entry.Value(), &check); err != nil {
					log.Printf("Warning: Failed to unmarshal settings for key %s: %v", entry.Key(), err)
					return
				}

				// Default check frequency is 60 seconds
				if check.Frequency == 0 {
					check.Frequency = 60
				}

				cron.AddCheckCron(cronScheduler, natsKVClient, checkName, check.Frequency)
				log.Printf("Check %s added", checkName)

			case "KeyValuePurgeOp":
			case "KeyValueDeleteOp":
				if !initialized {
					continue
				}
				cron.RemoveCheckCron(cronScheduler, checkName)
				log.Printf("Check %s removed", checkName)
			default:
				return
			}
		}
	}()
	log.Println("Watching KV for check changes")

	// Create OAuth2 config
	oauthConfig := &oauth2.Config{
		RedirectURL:  cfg.RedirectURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes: []string{
			"guilds",
			"guilds.members.read",
			"identify",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discord.com/api/oauth2/authorize",
			TokenURL:  "https://discord.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	// Create handler
	h := handlers.NewHandler(&handlers.Handler{
		OauthConfig:     oauthConfig,
		NatsAuthService: natsAuthService,
		NatsKVClient:    natsKVClient,
		TargetGuildID:   cfg.DiscordGuildID,
		RoleMap:         cfg.DiscordRoleMap,
		AccessTokens:    cfg.StaticAuthMap,
		State:           state,
		FrontendURL:     cfg.FrontendURL,
		CronScheduler:   cronScheduler,
	})

	// Create CORS middleware (allow frontend origin)
	corsMiddleware := middleware.NewCORSMiddleware([]string{cfg.FrontendURL, "http://localhost:5173"})

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(natsAuthService, cfg.DiscordRoleMap)

	// Register routes with CORS
	http.HandleFunc("/login", h.Login)
	http.HandleFunc("/auth/verify", corsMiddleware.Handler(h.Verify))
	http.HandleFunc("/auth/callback", h.Callback)
	http.HandleFunc("/auth/token", corsMiddleware.Handler(h.TokenLogin))
	http.HandleFunc("/api/checks/mutable-fields", corsMiddleware.Handler(authMiddleware.RequireAuth(h.GetMutableFields)))
	http.HandleFunc("/api/checks", corsMiddleware.Handler(h.Checks))
	http.HandleFunc("/api/settings", corsMiddleware.Handler(authMiddleware.RequireAuth(h.TeamSettings)))

	http.HandleFunc("/api/admin/settings", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(h.GetChecks)))
	http.HandleFunc("/api/admin/cron/start", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(h.StartScoringCron)))
	http.HandleFunc("/api/admin/cron/stop", corsMiddleware.Handler(authMiddleware.RequireAdminAuth(h.StopScoringCron)))

	// Create HTTP server with proper configuration
	server := &http.Server{
		Addr:         "0.0.0.0:3000",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Listening on :3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal
	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop KV watcher if running
	if watcher != nil {
		log.Println("Stopping KV watcher...")
		watcher.Stop()
	}

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")

	return nil
}
