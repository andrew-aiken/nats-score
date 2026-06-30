package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"server/pkg/auth"
	"server/pkg/config"
	"server/pkg/cron"
	"server/pkg/handlers"
	"server/pkg/logging"
	"server/pkg/middleware"
	"server/pkg/nats"

	"github.com/go-co-op/gocron/v2"
	natsnats "github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

type check struct {
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

type ServerArgs struct {
	LogLevel       string
	ConfigFilePath string
}

func Server(args ServerArgs) error {
	logging.SetupLogging(args.LogLevel)

	// Load configuration
	cfg, err := config.Load(args.ConfigFilePath)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		return nil
	}

	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		fmt.Printf("Failed to create cron scheduler: %v\n", err)
	}
	defer cronScheduler.Shutdown()

	// Initialize NATS auth service
	natsAuthService, err := auth.NewNATSAuthService(cfg.AccountSigningSeed, cfg.AccountPublicKey)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize NATS auth service: %v", err))
		return nil
	}

	// Initialize NATS KV client (optional - only if NATS URL is configured)
	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	err = natsClient.SetupConnection()
	if err != nil {
		slog.Error("Failed to initialize NATS KV client")
		return err
	} else {
		slog.Debug("Connected to NATS KV bucket 'settings'")
		defer natsClient.Close()
	}

	kv := natsClient.NatsKV
	watcher, err := kv.Watch("check.*")
	if err != nil {
		slog.Error("Failed to start KV watcher")
		return err
	}
	defer watcher.Stop()

	// Watch for check updates in background
	go monitorChecks(watcher, cronScheduler, natsClient.NatsConn)

	slog.Info("Watching KV for check changes")

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
		NatsKVClient:    natsClient.NatsKV,
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
		Addr:         "0.0.0.0:" + strconv.Itoa(cfg.HttpPort),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info(fmt.Sprintf("Listening on port %d", cfg.HttpPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error(fmt.Sprintf("Failed to start http server: %v", err))
			os.Exit(1) // Kill since we're in a gorouting, probably a cleaner way but this works
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt signal
	sig := <-sigChan
	slog.Info("Received shutting down trigger...", "signal", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop KV watcher if running
	if watcher != nil {
		slog.Debug("Stopping KV watcher")
		watcher.Stop()
	}

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error shutting down http server")
		return err
	}

	slog.Info("Server stopped")

	return nil
}

// monitorChecks monitors nats kv changes and schedules cronjobs on change
func monitorChecks(kvWatcher natsnats.KeyWatcher, cronScheduler gocron.Scheduler, natsConnection *natsnats.Conn) {
	initialized := false
	for entry := range kvWatcher.Updates() {
		if entry == nil {
			initialized = true
			continue
		}

		// Remove the prefix from the nats KV
		checkName := strings.TrimPrefix(entry.Key(), "check.")

		// Filter based on event type
		switch entry.Operation().String() {
		case "KeyValuePutOp":
			var check check

			if err := json.Unmarshal(entry.Value(), &check); err != nil {
				slog.Warn("Failed to unmarshal settings", entry.Key(), err)
				return
			}

			// Default check frequency is 60 seconds
			if check.Frequency == 0 {
				check.Frequency = 60
			}

			_, err := cron.AddCheckCron(cronScheduler, natsConnection, checkName, check.Frequency)
			if err != nil {
				slog.Warn("Failed to add check to cron", "error", err)
				return
			}
			slog.Info("Added check to cron", "name", checkName)

		case "KeyValuePurgeOp":
		case "KeyValueDeleteOp":
			if !initialized {
				continue
			}
			cron.RemoveCheckCron(cronScheduler, checkName)
			slog.Info("Removed check fro cron", "name", checkName)
		default:
			return
		}
	}
}
