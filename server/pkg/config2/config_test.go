package config

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

// mockKeyWatcher implements nats.KeyWatcher
type mockKeyWatcher struct {
	ch    chan nats.KeyValueEntry
	errCh chan error
}

func newMockWatcher() *mockKeyWatcher {
	return &mockKeyWatcher{
		ch:    make(chan nats.KeyValueEntry),
		errCh: make(chan error),
	}
}

func (m *mockKeyWatcher) Context() context.Context           { return context.Background() }
func (m *mockKeyWatcher) Updates() <-chan nats.KeyValueEntry { return m.ch }
func (m *mockKeyWatcher) Stop() error                        { return nil }
func (m *mockKeyWatcher) Error() <-chan error                { return m.errCh }

// mockKVEntry implements nats.KeyValueEntry
type mockKVEntry struct {
	key   string
	value []byte
	op    nats.KeyValueOp
}

func (m *mockKVEntry) Bucket() string             { return "settings" }
func (m *mockKVEntry) Key() string                { return m.key }
func (m *mockKVEntry) Value() []byte              { return m.value }
func (m *mockKVEntry) Revision() uint64           { return 1 }
func (m *mockKVEntry) Delta() uint64              { return 0 }
func (m *mockKVEntry) Created() time.Time         { return time.Now() }
func (m *mockKVEntry) Operation() nats.KeyValueOp { return m.op }

// runWithEvents sends entries to a Settings via MonitorSettings, then cancels.
// Uses an unbuffered channel so each send blocks until MonitorSettings receives it,
// guaranteeing all events are processed before cancellation.
func runWithEvents(settings *Settings, entries []nats.KeyValueEntry) {
	watcher := newMockWatcher()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for _, e := range entries {
			watcher.ch <- e
		}
		cancel()
	}()

	settings.MonitorSettings(ctx, watcher)
}

func TestMonitorSettings_CheckUpdate(t *testing.T) {
	checkJSON := `{
		"name": "noop-test",
		"type": "noop",
		"description": "test",
		"mutableFields": [],
		"scoreWeight": 5,
		"definition": {"pass": true}
	}`

	settings := &Settings{
		Checks: make(map[string]Check),
		Teams:  make(map[uint16]*TeamState),
	}

	runWithEvents(settings, []nats.KeyValueEntry{
		&mockKVEntry{key: "check.noop-test", value: []byte(checkJSON), op: nats.KeyValuePut},
	})

	check, ok := settings.Checks["noop-test"]
	if !ok {
		t.Fatal("check noop-test not found in settings")
	}
	if check.Name != "noop-test" {
		t.Errorf("Name = %q, want %q", check.Name, "noop-test")
	}
	if check.ScoreWeight != 5 {
		t.Errorf("ScoreWeight = %d, want 5", check.ScoreWeight)
	}
}

func TestMonitorSettings_TeamAttributeUpdate(t *testing.T) {
	attrs := map[string]map[string]string{
		"dns-check": {"host": "10.0.0.5"},
	}
	b, _ := json.Marshal(attrs)

	settings := &Settings{
		Checks: make(map[string]Check),
		Teams: map[uint16]*TeamState{
			5: {Attributes: make(map[string]map[string]string)},
		},
	}

	runWithEvents(settings, []nats.KeyValueEntry{
		&mockKVEntry{key: "5.settings", value: b, op: nats.KeyValuePut},
	})

	got := settings.Teams[5].Attributes["dns-check"]["host"]
	if got != "10.0.0.5" {
		t.Errorf("host = %q, want %q", got, "10.0.0.5")
	}
}

func TestMonitorSettings_OtherTeamNotUpdated(t *testing.T) {
	attrs := map[string]map[string]string{
		"dns-check": {"host": "10.0.0.5"},
	}
	b, _ := json.Marshal(attrs)

	settings := &Settings{
		Checks: make(map[string]Check),
		Teams: map[uint16]*TeamState{
			5:  {Attributes: make(map[string]map[string]string)},
			10: {Attributes: make(map[string]map[string]string)},
		},
	}

	// Send update for team 5 only; team 10 should be untouched
	runWithEvents(settings, []nats.KeyValueEntry{
		&mockKVEntry{key: "5.settings", value: b, op: nats.KeyValuePut},
	})

	if len(settings.Teams[10].Attributes) != 0 {
		t.Errorf("team 10 attributes should be empty, got %v", settings.Teams[10].Attributes)
	}
}

func TestMonitorSettings_DeleteIgnored(t *testing.T) {
	attrs := map[string]map[string]string{
		"dns-check": {"host": "original"},
	}
	settings := &Settings{
		Checks: make(map[string]Check),
		Teams: map[uint16]*TeamState{
			5: {Attributes: attrs},
		},
	}

	// A delete operation should be skipped
	runWithEvents(settings, []nats.KeyValueEntry{
		&mockKVEntry{key: "5.settings", value: nil, op: nats.KeyValueDelete},
	})

	if settings.Teams[5].Attributes["dns-check"]["host"] != "original" {
		t.Error("attributes should not have changed on delete")
	}
}

func TestMonitorSettings_NilEntryHandled(t *testing.T) {
	settings := &Settings{
		Checks: make(map[string]Check),
		Teams:  make(map[uint16]*TeamState),
	}

	// A nil entry (initial sync marker) should not panic
	runWithEvents(settings, []nats.KeyValueEntry{nil})
}

// TestMonitorSettings_DataRace reproduces the race between MonitorSettings (writer)
// and concurrent map reads that mirror what HandleScoreEvent does on each NATS message.
// Run with: go test -race -run TestMonitorSettings_DataRace ./pkg/config/
func TestMonitorSettings_DataRace(t *testing.T) {
	checkJSON := `{
		"name": "noop-race",
		"type": "noop",
		"description": "race test",
		"mutableFields": [],
		"scoreWeight": 1,
		"definition": {"pass": true}
	}`

	settings := &Settings{
		Checks: make(map[string]Check),
		Teams:  make(map[uint16]*TeamState),
	}

	watcher := newMockWatcher()
	ctx := t.Context()

	// MonitorSettings runs in the background, continuously writing to settings.Checks
	go settings.MonitorSettings(ctx, watcher)

	// Feed a steady stream of KV updates to drive concurrent writes
	go func() {
		entry := &mockKVEntry{
			key:   "check.noop-race",
			value: []byte(checkJSON),
			op:    nats.KeyValuePut,
		}
		for {
			select {
			case <-ctx.Done():
				return
			case watcher.ch <- entry:
			}
		}
	}()

	// Simulate HandleScoreEvent: read settings concurrently with the writes above.
	// Uses the accessor so both sides go through the mutex — this should not race.
	deadline := time.After(100 * time.Millisecond)
	for {
		select {
		case <-deadline:
			return
		default:
			_, _ = settings.GetCheck("noop-race")
		}
	}
}

func TestCheckUnmarshalJSON(t *testing.T) {
	t.Run("valid noop check", func(t *testing.T) {
		data := []byte(`{
			"name": "my-noop",
			"type": "noop",
			"description": "a noop check",
			"mutableFields": ["Pass"],
			"scoreWeight": 10,
			"definition": {"pass": false}
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Name != "my-noop" {
			t.Errorf("Name = %q, want %q", c.Name, "my-noop")
		}
		if c.ScoreWeight != 10 {
			t.Errorf("ScoreWeight = %d, want 10", c.ScoreWeight)
		}
		if c.Definition == nil {
			t.Error("Definition should not be nil")
		}
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "unknown-type",
			"definition": {}
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for unknown type")
		}
	})

	t.Run("missing definition returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "noop"
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for missing definition")
		}
	})

	t.Run("null definition returns error", func(t *testing.T) {
		data := []byte(`{
			"name": "bad",
			"type": "noop",
			"definition": null
		}`)

		var c Check
		if err := json.Unmarshal(data, &c); err == nil {
			t.Error("expected error for null definition")
		}
	})
}
