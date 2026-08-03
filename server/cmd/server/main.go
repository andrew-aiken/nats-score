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

	"server/internal/auth"
	"server/internal/config"
	"server/internal/cron"
	"server/internal/handlers"
	"server/internal/logging"
	"server/internal/middleware"
	"server/internal/nats"
	"server/internal/routes"

	"github.com/go-co-op/gocron/v2"
	natsnats "github.com/nats-io/nats.go"
)

type check struct {
	Frequency   int16  `json:"frequency"`   // How often the check runs in seconds
	Type        string `json:"type"`        // Type of check
	Description string `json:"description"` // Additional information about the check
	ScoreWeight int8   `json:"scoreWeight"` // How many points to assign the check
}

type ServerArgs struct {
	LogLevel       string
	ConfigFilePath string
}

// Server setups the initial connection to NATS, watches checks that get loaded into cron, and runs the webserver
func Server(args ServerArgs) error {
	logging.SetupLogging(args.LogLevel)

	// Load configuration
	cfg, err := config.Load(args.ConfigFilePath)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to load configuration: %v", err))
		return fmt.Errorf("load config: %w", err)
	}

	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create cron scheduler: %v", err))
		return fmt.Errorf("create cron scheduler: %w", err)
	}
	defer cronScheduler.Shutdown()

	// Initialize NATS auth service
	natsAuthService, err := auth.NewNATSAuthService(cfg.AccountSigningSeed, cfg.AccountPublicKey)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize NATS auth service: %v", err))
		return fmt.Errorf("initialize NATS auth service: %w", err)
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
		slog.Debug("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	if err := natsClient.SetupUsersKV(); err != nil {
		slog.Error("Failed to initialize NATS users KV client")
		return err
	}

	kv := natsClient.NatsKV
	kvWatcher, err := kv.Watch("check.*")
	if err != nil {
		slog.Error("Failed to start KV watcher")
		return err
	}
	defer kvWatcher.Stop()

	// Watch for check updates in background
	go monitorChecks(kvWatcher, cronScheduler, natsClient.NatsConn)
	slog.Info("Watching KV for check changes")

	// Create handler
	h := handlers.NewHandler(&handlers.Handler{
		NatsAuthService:   natsAuthService,
		NatsKVClient:      natsClient.NatsKV,
		NatsUsersKVClient: natsClient.NatsUsersKV,
		CronScheduler:     cronScheduler,
	})

	authMiddleware := middleware.NewAuthMiddleware(natsAuthService)
	corsMiddleware := middleware.NewCORSMiddleware([]string{cfg.FrontendURL, "http://localhost:5173"})

	mux := routes.SetupRoutes(h, *corsMiddleware, *authMiddleware)

	server := &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(cfg.HttpPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrCh := make(chan error, 1)

	// Start server in the background
	go func() {
		serverErrCh <- routes.StartServer(server)
	}()
	slog.Info(fmt.Sprintf("Started webserver, listening on port %d", cfg.HttpPort))

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
		return nil
	case sig := <-sigChan:
		slog.Info("Received shutting down trigger...", "signal", sig)
	}

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop KV watcher if running
	if kvWatcher != nil {
		slog.Debug("Stopping KV watcher")
		kvWatcher.Stop()
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
				continue
			}

			// Default check frequency is 60 seconds
			if check.Frequency == 0 {
				check.Frequency = 60
			}

			_, err := cron.AddCheckCron(cronScheduler, natsConnection, checkName, check.Frequency)
			if err != nil {
				slog.Warn("Failed to add check to cron", "error", err)
				continue
			}
			slog.Info("Added check to cron", "name", checkName)

		case "KeyValuePurgeOp":
		case "KeyValueDeleteOp":
			if !initialized {
				continue
			}
			cron.RemoveCheckCron(cronScheduler, checkName)
			slog.Info("Removed check from cron", "name", checkName)
		default:
			slog.Warn("Ignoring unknown KV operation", "operation", entry.Operation().String(), "key", entry.Key())
			continue
		}
	}
}
