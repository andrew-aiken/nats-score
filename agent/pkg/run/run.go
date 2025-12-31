package run

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aaiken/nats-score/pkg/config"
	"github.com/aaiken/nats-score/pkg/nats"
)

type RunArgs struct {
	NatsUrl       string
	NatsCredsFile string
	TeamNumber    int16
}

func Run(args RunArgs) {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var natsCon = nats.NatsConnection{
		NatsUrl:            args.NatsUrl,
		NatsConnectionName: fmt.Sprintf("%d-agent", args.TeamNumber),
		NatsCredsFile:      args.NatsCredsFile,
	}

	err := natsCon.SetupConnection()
	if err != nil {
		log.Fatalln(err)
	}

	// Watch global settings and team-specific settings
	watchList := []string{"settings", fmt.Sprintf("%d.settings", args.TeamNumber)}
	if err = natsCon.SetupKVWatcher(watchList); err != nil {
		log.Fatalf("Failed to start KV watcher: %v", err)
	}

	var agentSettings config.Settings
	agentSettings.StaticConf.TeamNumber = args.TeamNumber

	if err = natsCon.SubjectSubscribe(&agentSettings); err != nil {
		log.Fatalf("%v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
		natsCon.Close()
	}()

	// Process KV settings updates
	agentSettings.MonitorSettings(ctx, fmt.Sprint(args.TeamNumber), natsCon.NatsKVWatcher)
}
