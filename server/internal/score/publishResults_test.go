package score

import (
	"errors"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"

	"github.com/andrew-aiken/checks"
)

func TestPublishResults(t *testing.T) {
	streamName := "dummyStream"
	var scoreWeight uint8 = 8

	results := checks.Results{
		Message:   "dummy",
		Passed:    true,
		Timestamp: time.Now(),
	}

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	nc, err := nats.Connect(server.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to initialize JetStream: %v", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{fmt.Sprintf("%s.*", streamName)},
	})
	if err != nil {
		t.Error("Failed to create test nats stream")
	}

	t.Run("expected results", func(t *testing.T) {
		err = publishResults(streamName, results, scoreWeight, js)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("non-existent stream", func(t *testing.T) {
		streamName := "dneStream"
		err = publishResults(streamName, results, scoreWeight, js)

		if errors.Is(err, nats.ErrStreamNotFound) {
			t.Error(err)
		}
	})
}
