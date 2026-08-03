package nats

import (
	"context"
	"errors"
	"testing"
	"time"

	natsio "github.com/nats-io/nats.go"
)

// --- mocks ---

type mockKeyWatcher struct {
	ch    chan natsio.KeyValueEntry
	errCh chan error
}

func (m *mockKeyWatcher) Context() context.Context             { return context.Background() }
func (m *mockKeyWatcher) Updates() <-chan natsio.KeyValueEntry { return m.ch }
func (m *mockKeyWatcher) Stop() error                          { return nil }
func (m *mockKeyWatcher) Error() <-chan error                  { return m.errCh }

type mockKeyValue struct {
	watchFilteredErr error
	watchFilteredKW  natsio.KeyWatcher
}

func (m *mockKeyValue) WatchFiltered(keys []string, opts ...natsio.WatchOpt) (natsio.KeyWatcher, error) {
	return m.watchFilteredKW, m.watchFilteredErr
}

// Unused KeyValue interface methods required to satisfy the interface.
func (m *mockKeyValue) Get(key string) (natsio.KeyValueEntry, error) { return nil, nil }
func (m *mockKeyValue) GetRevision(key string, revision uint64) (natsio.KeyValueEntry, error) {
	return nil, nil
}
func (m *mockKeyValue) Put(key string, value []byte) (uint64, error)                 { return 0, nil }
func (m *mockKeyValue) PutString(key string, value string) (uint64, error)           { return 0, nil }
func (m *mockKeyValue) Create(key string, value []byte) (uint64, error)              { return 0, nil }
func (m *mockKeyValue) Update(key string, value []byte, last uint64) (uint64, error) { return 0, nil }
func (m *mockKeyValue) Delete(key string, opts ...natsio.DeleteOpt) error            { return nil }
func (m *mockKeyValue) Purge(key string, opts ...natsio.DeleteOpt) error             { return nil }
func (m *mockKeyValue) Watch(keys string, opts ...natsio.WatchOpt) (natsio.KeyWatcher, error) {
	return nil, nil
}
func (m *mockKeyValue) WatchAll(opts ...natsio.WatchOpt) (natsio.KeyWatcher, error) { return nil, nil }
func (m *mockKeyValue) Keys(opts ...natsio.WatchOpt) ([]string, error)              { return nil, nil }
func (m *mockKeyValue) ListKeys(opts ...natsio.WatchOpt) (natsio.KeyLister, error)  { return nil, nil }
func (m *mockKeyValue) History(key string, opts ...natsio.WatchOpt) ([]natsio.KeyValueEntry, error) {
	return nil, nil
}
func (m *mockKeyValue) Bucket() string                             { return "settings" }
func (m *mockKeyValue) PurgeDeletes(opts ...natsio.PurgeOpt) error { return nil }
func (m *mockKeyValue) Status() (natsio.KeyValueStatus, error)     { return nil, nil }

// --- tests ---

func TestClose_EmptyConnection(t *testing.T) {
	n := NatsConnection{}
	n.Close() // must not panic
}

func TestClose_Idempotent(t *testing.T) {
	n := NatsConnection{}
	n.Close()
	n.Close() // second call must not panic
}

func TestSetupKVWatcher_PropagatesError(t *testing.T) {
	n := NatsConnection{
		NatsKV: &mockKeyValue{watchFilteredErr: errors.New("watcher setup failed")},
	}

	err := n.SetupKVWatcher([]string{"check.*"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSetupKVWatcher_StoresWatcher(t *testing.T) {
	watcher := &mockKeyWatcher{
		ch:    make(chan natsio.KeyValueEntry),
		errCh: make(chan error),
	}
	n := NatsConnection{
		NatsKV: &mockKeyValue{watchFilteredKW: watcher},
	}

	if err := n.SetupKVWatcher([]string{"check.*", "5.settings"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.NatsKVWatcher == nil {
		t.Error("NatsKVWatcher should be set after successful SetupKVWatcher")
	}
}

// mockKeyWatcher field not used by nats_test but needed by config_test mock pattern;
// keep time import used.
var _ = time.Now
