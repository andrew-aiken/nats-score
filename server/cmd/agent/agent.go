package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/andrew-aiken/checks"

	nats "server/pkg/nats2"
	config "server/pkg/config2"
)

type RunArgs struct {
	LogLevel      string
	NatsUrl       string
	NatsCredsFile string
	TeamNumbers   []uint16
}

func Run(args RunArgs) error {
	setupLogging(args.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var connName string
	if len(args.TeamNumbers) == 1 {
		connName = fmt.Sprintf("%d-agent", args.TeamNumbers[0])
	} else {
		connName = "multi-agent"
	}

	var natsCon = nats.NatsConnection{
		NatsUrl:            args.NatsUrl,
		NatsConnectionName: connName,
		NatsCredsFile:      args.NatsCredsFile,
		NatsInboxPrefix:    "_INBOX.0",
	}

	err := natsCon.SetupConnection()
	if err != nil {
		return err
	}
	defer natsCon.Close()

	watchList := []string{"check.*"}
	for _, n := range args.TeamNumbers {
		watchList = append(watchList, fmt.Sprintf("%d.settings", n))
		if args.LogLevel == "debug" {
			slog.Debug("Team Loaded", "team", n)
		}
	}

	if err = natsCon.SetupKVWatcher(watchList); err != nil {
		return err
	}

	var agentSettings config.Settings
	agentSettings.Checks = make(map[string]config.Check)
	agentSettings.Teams = make(map[uint16]*config.TeamState, len(args.TeamNumbers))
	for _, n := range args.TeamNumbers {
		agentSettings.Teams[n] = &config.TeamState{
			Attributes: make(map[string]map[string]string),
			StaticConf: checks.StaticConf{
				TeamNumber:    n,
				TeamNumberHex: fmt.Sprintf("%x", n),
			},
		}
	}

	if err = natsCon.SubjectSubscribe(ctx, &agentSettings); err != nil {
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
	agentSettings.MonitorSettings(ctx, args.TeamNumbers, natsCon.NatsKVWatcher)

	return nil
}

// ParseTeams parses a string like "1,3,5-8,10" into a slice of uint16.
// Individual values and inclusive ranges (a-b) are supported and may be combined.
func ParseTeams(s string) ([]uint16, error) {
	var result []uint16
	seen := make(map[uint16]struct{})

	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if lo, hi, found := strings.Cut(part, "-"); found {
			loN, err := strconv.ParseInt(strings.TrimSpace(lo), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q: %w", lo, err)
			}
			hiN, err := strconv.ParseInt(strings.TrimSpace(hi), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q: %w", hi, err)
			}
			if loN > hiN {
				return nil, fmt.Errorf("range %q has start greater than end", part)
			}
			for n := loN; n <= hiN; n++ {
				t := uint16(n)
				if _, dup := seen[t]; !dup {
					seen[t] = struct{}{}
					result = append(result, t)
				}
			}
		} else {
			n, err := strconv.ParseInt(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid team number %q: %w", part, err)
			}
			t := uint16(n)
			if _, dup := seen[t]; !dup {
				seen[t] = struct{}{}
				result = append(result, t)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no teams specified")
	}
	return result, nil
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
