package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"server/internal/logging"
	"server/internal/nats"

	"github.com/andrew-aiken/checks/helper"

	"github.com/creasty/defaults"
	natsnats "github.com/nats-io/nats.go"
)

// Check is a standardized form of input types for consumers
// TODO: This in theory could be using the settings.Check but would require getting the check type and then unmarshal the definition
type check struct {
	// Parameters for the check
	Definition json.RawMessage `json:"definition"`
	// Additional information about the check
	Description string `json:"description"`
	// How often the check runs in seconds
	Frequency uint16 `json:"frequency"`
	// Fields in the definition that can be overwritten
	MutableFields []string `json:"mutableFields"`
	// Name of the check
	Name string `json:"name"`
	// How many points to assign the check
	ScoreWeight uint8 `json:"scoreWeight"`
	// Type of check
	Type string `json:"type"`
}

// Validate checks if a check is in a valid structure
func Validate(natsAddress string, natsCreds string, checkName string) error {
	logging.SetupLogging("warn")

	if checkName == "" {
		fmt.Println("Check name required")
		return nil
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       natsAddress,
		NatsCredsFile: natsCreds,
	}
	err := natsClient.SetupConnection()
	if err != nil {
		slog.Warn("Failed to initialize NATS KV client")
		return err
	} else {
		slog.Debug("Connected to NATS KV bucket")
		defer natsClient.Close()
	}

	// Get NATS key value handler
	kv := natsClient.NatsKV

	checkKey := fmt.Sprintf("check.%s", checkName)

	key, err := kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		return errors.New("check does not exist")
	} else if err != nil {
		return err
	}

	data := key.Value()

	var check check
	if err = json.Unmarshal(data, &check); err != nil {
		return err
	}

	if err := validateCheck(check); err != nil {
		return err
	}

	// Skip unmarshalling if definition is empty or null
	if len(check.Definition) == 0 || string(check.Definition) == "null" {
		return fmt.Errorf("definition not defined")
	}

	checkDefinition, err := helper.NewDefinition(check.Type)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(check.Definition, checkDefinition); err != nil {
		return fmt.Errorf("failed to unmarshal definition type %q: %w", check.Type, err)
	}

	err = defaults.Set(checkDefinition)
	if err != nil {
		return err
	}

	type Validator interface {
		Validate() (bool, string)
	}

	v, ok := checkDefinition.(Validator)
	if !ok {
		return fmt.Errorf("check type %q does not implement Validate()", check.Type)
	}

	passed, msg := v.Validate()
	if passed {
		fmt.Println("Check validation passed")
	} else {
		return fmt.Errorf("check validation failed, error: %s", msg)
	}

	return nil
}

func validateCheck(check check) error {
	if check.Frequency == 0 {
		slog.Warn("Frequency undefined, will default to 60 seconds")
	}

	if check.Name == "" {
		slog.Warn("The checks name not defined, should be set to the filename to reduct confusion")
	}

	if check.ScoreWeight == 0 {
		return fmt.Errorf("checks weight is not defined")
	}

	if check.Type == "" {
		return fmt.Errorf("check type not defined")
	}
	return nil
}
