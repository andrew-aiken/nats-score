package checks_test

import (
	"strings"
	"testing"

	"server/cmd/checks"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
)

func TestValidate(t *testing.T) {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.StoreDir = t.TempDir()

	server := natsserver.RunServer(&opts)
	defer server.Shutdown()

	natsServerAddress := server.Addr().String()
	checkName := "noop"

	nc, err := nats.Connect(natsServerAddress)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to connect to JetStream: %v", err)
	}

	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
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

	t.Run("NoCheck", func(t *testing.T) {
		err := checks.Validate(natsServerAddress, "", "")
		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "check name required") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoNats", func(t *testing.T) {
		err := checks.Validate("DNE", "", checkName)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "failed to connect to NATS after 3 attempts: dial tcp: lookup DNE: no such host") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("NoCheck", func(t *testing.T) {
		err := checks.Validate(natsServerAddress, "", checkName)

		if err == nil {
			t.Fatal("Expected error")
		}

		if !strings.Contains(err.Error(), "check does not exist") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("BadJson", func(t *testing.T) {
		_, err = kv.Put("check.noop", []byte("{]"))
		if err != nil {
			t.Fatal(err.Error())
		}

		err = checks.Validate(natsServerAddress, "", checkName)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "invalid character ']' looking for beginning of object key string") {
			t.Fatalf("Got wrong error message: '%v'", err.Error())
		}
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("Weight", func(t *testing.T) {
			_, err = kv.Put("check.noop", []byte("{}"))
			if err != nil {
				t.Fatal(err.Error())
			}

			err = checks.Validate(natsServerAddress, "", checkName)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), "checks weight is not defined") {
				t.Fatalf("Got wrong error message: '%v'", err.Error())
			}
		})

		t.Run("Type", func(t *testing.T) {
			_, err = kv.Put("check.noop", []byte(`{"scoreWeight": 1}`))
			if err != nil {
				t.Fatal(err.Error())
			}

			err = checks.Validate(natsServerAddress, "", checkName)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), "check type not defined") {
				t.Fatalf("Got wrong error message: '%v'", err.Error())
			}
		})
	})


	t.Run("Definition", func(t *testing.T) {
		t.Run("Missing", func(t *testing.T) {
			_, err = kv.Put("check.noop", []byte(`{"scoreWeight": 1, "type": "noop"}`))
			if err != nil {
				t.Fatal(err.Error())
			}

			err = checks.Validate(natsServerAddress, "", checkName)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), "definition not defined") {
				t.Fatalf("Got wrong error message: '%v'", err.Error())
			}
		})

		t.Run("UnknownType", func(t *testing.T) {
			_, err = kv.Put("check.noop", []byte(`{"scoreWeight": 1, "type": "dne", "definition": {}}`))
			if err != nil {
				t.Fatal(err.Error())
			}

			err = checks.Validate(natsServerAddress, "", checkName)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), `unknown check type "dne"`) {
				t.Fatalf("Got wrong error message: '%v'", err.Error())
			}
		})

		t.Run("BadDefinition", func(t *testing.T) {
			_, err = kv.Put("check.noop", []byte(`{"scoreWeight": 1, "type": "noop", "definition": []}`))
			if err != nil {
				t.Fatal(err.Error())
			}

			err = checks.Validate(natsServerAddress, "", checkName)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), `failed to unmarshal definition type "noop": json: cannot unmarshal array into Go value of type noop.Definition`) {
				t.Fatalf("Got wrong error message: '%v'", err.Error())
			}
		})
	})
}
