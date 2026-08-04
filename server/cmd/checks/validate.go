package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"server/internal/config"
	"server/internal/logging"
	"server/internal/nats"

	"github.com/andrew-aiken/checks/helper"

	"github.com/creasty/defaults"
	natsnats "github.com/nats-io/nats.go"
)

// TODO have this struct centralized
type Check struct {
	Definition    json.RawMessage `json:"definition"`    // Parameters for the check
	Description   string          `json:"description"`   // Additional information about the check
	Frequency     uint16          `json:"frequency"`     // How often the check runs in seconds
	MutableFields []string        `json:"mutableFields"` // Fields in the definition that can be overwritten
	Name          string          `json:"name"`          // Name of the check
	ScoreWeight   uint8           `json:"scoreWeight"`   // How many points to assign the check
	Type          string          `json:"type"`          // Type of check
}

func Validate(configFile string, checkName string) error {
	logging.SetupLogging("warn")

	if checkName == "" {
		fmt.Println("Check name required")
		return nil
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config")
		return err
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	err = natsClient.SetupConnection()
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
		return errors.New("Check does not exist")
	} else if err != nil {
		return err
	}

	data := key.Value()

	var check Check
	if err = json.Unmarshal(data, &check); err != nil {
		return err
	}

	if err := validateCheck(check); err != nil {
		return err
	}

	// Skip unmarshalling if definition is empty or null
	if len(check.Definition) == 0 || string(check.Definition) == "null" {
		return fmt.Errorf("Definition not defined")
	}

	checkDefinition, err := helper.NewDefinition(check.Type)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(check.Definition, checkDefinition); err != nil {
		return fmt.Errorf("failed to unmarshal definition type %q: %w", check.Type, err)
	}

	defaults.Set(checkDefinition)

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
		return fmt.Errorf("Check validation failed, error: %s", msg)
	}

	return nil
}

func validateCheck(check Check) error {
	if check.Frequency == 0 {
		slog.Warn("Frequency undefined, will default to 60 seconds")
	}

	if check.Name == "" {
		slog.Warn("The checks name not defined, should be set to the filename to reduct confusion")
	}

	if check.ScoreWeight == 0 {
		return fmt.Errorf("The checks weight is not defined.")
	}

	if check.Type == "" {
		return fmt.Errorf("Check type not defined")
	}
	return nil
}
