package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/aaiken/nats-score/pkg/config"
	"github.com/aaiken/nats-score/pkg/nats"
)

type RunArgs struct {
	LogLevel      string
	NatsUrl       string
	NatsCredsFile string
	TeamNumber    int16
}

func Run(args RunArgs) error {
	setupLogging(args.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var natsCon = nats.NatsConnection{
		NatsUrl:            args.NatsUrl,
		NatsConnectionName: fmt.Sprintf("%d-agent", args.TeamNumber),
		NatsCredsFile:      args.NatsCredsFile,
	}

	err := natsCon.SetupConnection()
	if err != nil {
		return err
	}

	// Watch global settings and team-specific settings
	watchList := []string{
		"check.*",
		fmt.Sprintf("%d.settings", args.TeamNumber), // TODO change to settings.X
	}

	if err = natsCon.SetupKVWatcher(watchList); err != nil {
		return fmt.Errorf("Failed to start KV watcher: %v", err)
	}

	var agentSettings config.Settings
	agentSettings.StaticConf.TeamNumber = args.TeamNumber

	if err = natsCon.SubjectSubscribe(&agentSettings); err != nil {
		return err
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
		natsCon.Close()
	}()

	// Process KV settings updates
	agentSettings.MonitorSettings(ctx, fmt.Sprint(args.TeamNumber), natsCon.NatsKVWatcher)

	return nil
}

func setupLogging(logLevel string) {
	var slogLevel slog.Level

	switch strings.ToLower(logLevel) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source, _ := a.Value.Any().(*slog.Source)
				if source != nil {
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	}))

	slog.SetDefault(logger)
}
