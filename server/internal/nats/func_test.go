package nats_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/andrew-aiken/score/internal/nats"
	"github.com/andrew-aiken/score/internal/settings"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsnats "github.com/nats-io/nats.go"
)

func TestChecks(t *testing.T) {
	checkName := "dummy"

	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	nc, err := natsnats.Connect(server.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	bucket, err := js.CreateKeyValue(&natsnats.KeyValueConfig{
		Bucket:       "settings",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatal(err)
	}

	kv, err := js.KeyValue(bucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	// Define a fake noop check object
	testCheck := settings.Check{
		Name: "dummyCheck",
		Definition: struct{ Pass bool }{
			Pass: true,
		},
		Description: "example definition",
		Frequency:   30,
		ScoreWeight: 1,
		Type:        "noop",
		MutableFields: []string{
			"pass",
		},
	}

	testCheckBytes, err := json.Marshal(testCheck)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}
	_, err = kv.Put("check."+checkName, testCheckBytes)
	if err != nil {
		t.Fatalf("Failed to insert test check: %v", err)
	}

	t.Run("GetChecks", func(t *testing.T) {
		checks, err := nats.GetChecks(kv)
		if err != nil {
			t.Fatalf("Failed to get checks: %v", err)
		}

		if len(checks) != 1 {
			t.Fatal("Expected exactly one check")
		}

		if checks[checkName].Name != testCheck.Name {
			t.Fatal("Check name does not match")
		}
	})

	t.Run("GetMutableFields", func(t *testing.T) {
		mutableFields, err := nats.GetMutableFields(kv)
		if err != nil {
			t.Fatalf("Failed to list mutable fields: %v", err)
		}

		if !slices.Equal(mutableFields[checkName], testCheck.MutableFields) {
			t.Fatal("Expected value of mutable fields does not match")
		}
	})

	// Running this last since it will break other tests
	t.Run("MalformedChecks", func(t *testing.T) {
		// Insert badly formatted user value
		_, err = kv.Put("check.badcheck", []byte("foo"))
		if err != nil {
			t.Fatalf("Failed to put bad check: %v", err)
		}

		_, err := nats.GetChecks(kv)
		unwrappedError := errors.Unwrap(err)
		if _, ok := errors.AsType[*json.SyntaxError](unwrappedError); !ok {
			t.Fatalf("Got incorrect error when unwrapped: %T", unwrappedError)
		}
	})
}

func TestTeamSettings(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	nc, err := natsnats.Connect(server.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	bucket, err := js.CreateKeyValue(&natsnats.KeyValueConfig{
		Bucket:       "settings",
		Description:  "Check & configuration storage",
		History:      5,
		TTL:          0,
		MaxValueSize: -1,
		MaxBytes:     -1,
	})
	if err != nil {
		t.Fatal(err)
	}

	kv, err := js.KeyValue(bucket.Bucket())
	if err != nil {
		t.Fatalf("Failed to connect to key value: %v", err)
	}

	teamNumber := "99"

	settings := map[string]map[string]string{
		"checkName": {
			"keyName": "value",
		},
	}

	t.Run("GetEmptyTeamSettings", func(t *testing.T) {
		checks, err := nats.GetTeamSettings(kv, teamNumber)
		if err != nil {
			t.Fatalf("Failed to get team settings: %v", err)
		}
		if len(checks) != 0 {
			t.Fatal("Expected zero length of checks map")
		}
	})

	t.Run("PutTeamSettings", func(t *testing.T) {
		err = nats.PutTeamSettings(kv, teamNumber, settings)
		if err != nil {
			t.Fatalf("Failed to put team settings: %v", err)
		}
	})

	t.Run("GetTeamSettings", func(t *testing.T) {
		_, err := nats.GetTeamSettings(kv, teamNumber)
		if err != nil {
			t.Fatalf("Failed to get team settings: %v", err)
		}
	})

	t.Run("GetInvalidTeamSettings", func(t *testing.T) {
		data, err := json.Marshal(map[string]string{
			"checkName": "wrong",
		})
		if err != nil {
			t.Fatalf("Failed to marshal invalid team settings: %v", err)
		}

		// Insert badly formatted user value
		_, err = kv.Put("00.settings", data)
		if err != nil {
			t.Fatalf("Failed to put bad user: %v", err)
		}

		_, err = nats.GetTeamSettings(kv, "00")
		unwrappedError := errors.Unwrap(err)
		if _, ok := errors.AsType[*json.UnmarshalTypeError](unwrappedError); !ok {
			t.Fatalf("Got incorrect error when unwrapped: %T", unwrappedError)
		}
	})
}
