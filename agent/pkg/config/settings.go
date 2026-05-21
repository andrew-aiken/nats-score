package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/andrew-aiken/checks"
	"github.com/nats-io/nats.go"
)

// TeamState contains information about an individuals team config
type TeamState struct {
	mu         sync.RWMutex
	Attributes map[string]map[string]string
	StaticConf checks.StaticConf
}

// GetAttributes returns the override attributes for a single check, safe for concurrent use.
func (ts *TeamState) GetAttributes(checkName string) map[string]string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.Attributes[checkName]
}

type Settings struct {
	mu     sync.RWMutex
	Checks map[string]Check
	Teams  map[uint16]*TeamState
}

// GetCheck returns the named check, safe for concurrent use.
func (s *Settings) GetCheck(name string) (Check, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.Checks[name]
	return c, ok
}

// MonitorSettings loops monitoring the nats KV for settings updates
func (s *Settings) MonitorSettings(ctx context.Context, teams []uint16, natsKVWatcher nats.KeyWatcher) {
	teamKeys := make(map[string]uint16, len(teams))
	for _, n := range teams {
		teamKeys[fmt.Sprintf("%d.settings", n)] = n
	}

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context cancelled, stopping watcher")
			return
		case entry := <-natsKVWatcher.Updates():
			if entry == nil {
				// Initial sync complete
				slog.Info("Initial KV sync complete, watching for updates...")
				continue
			}

			// Only process add/update operations, skip deletes and purges
			if entry.Operation() != nats.KeyValuePut {
				continue
			}

			key := entry.Key()
			checkName, isCheck := strings.CutPrefix(key, "check.")
			value := entry.Value()

			switch {
			case isCheck:
				slog.Info(fmt.Sprintf("Updating check %s", checkName))

				check := Check{}
				if err := json.Unmarshal(value, &check); err != nil {
					slog.Warn(fmt.Sprintf("Failed to unmarshal settings for check %s: %v", checkName, err))
					continue
				}

				s.mu.Lock()
				s.Checks[checkName] = check
				s.mu.Unlock()
			default:
				if teamID, ok := teamKeys[key]; ok {
					slog.Info(fmt.Sprintf("Team %d settings update", teamID))

					var teamSettings map[string]map[string]string
					if err := json.Unmarshal(value, &teamSettings); err != nil {
						slog.Warn(fmt.Sprintf("Failed to unmarshal settings for setting %s: %v", key, err))
						continue
					}

					if ts, exists := s.Teams[teamID]; exists {
						ts.mu.Lock()
						ts.Attributes = teamSettings
						ts.mu.Unlock()
					}
				}
			}
		}
	}
}
