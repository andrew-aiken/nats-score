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

	var startUpCompleted bool = false

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context cancelled, stopping watcher")
			return
		case entry := <-natsKVWatcher.Updates():
			if entry == nil {
				// Initial sync complete
				slog.Info("Initial KV sync complete, watching for updates...")
				startUpCompleted = true
				continue
			}

			key := entry.Key()
			value := entry.Value()

			checkName, isCheck := strings.CutPrefix(key, "check.")

			switch {
			case isCheck:
				// If the check is removed drop from the list of loaded checks and continue
				if entry.Operation() != nats.KeyValuePut {
					// If initial startup has not completed skip removing checks
					if !startUpCompleted {
						continue
					}

					s.mu.Lock()
					delete(s.Checks, checkName)
					s.mu.Unlock()
					slog.Info(fmt.Sprintf("Removed check %s", checkName))
					continue
				}

				slog.Info(fmt.Sprintf("Loading check %s", checkName))

				check := Check{}
				if err := json.Unmarshal(value, &check); err != nil {
					slog.Warn(fmt.Sprintf("Failed to unmarshal settings for check %s: %v", checkName, err))
					continue
				}

				s.mu.Lock()
				s.Checks[checkName] = check
				s.mu.Unlock()
			default:
				// TODO: Refactor
				if teamID, ok := teamKeys[key]; ok {
					if ts, exists := s.Teams[teamID]; exists {
						// If the nats change updates the setting update the stored value
						if entry.Operation() == nats.KeyValuePut {
							var teamSettings map[string]map[string]string
							if err := json.Unmarshal(value, &teamSettings); err != nil {
								slog.Warn(fmt.Sprintf("Failed to unmarshal settings for setting %s: %v", key, err))
								continue
							}

							ts.mu.Lock()
							ts.Attributes = teamSettings
							ts.mu.Unlock()
							slog.Info(fmt.Sprintf("Team %d settings update", teamID))
						} else { // If not updating its removing: drop the attributes key
							// If initial startup has not completed skip removing checks
							if !startUpCompleted {
								continue
							}

							s.mu.Lock()
							ts.Attributes = nil
							s.mu.Unlock()
							slog.Info(fmt.Sprintf("Removed settings for team %d", teamID))
						}
					}
				} else {
					slog.Warn(fmt.Sprintf("Unknown team key %v", key))
				}
			}
		}
	}
}
