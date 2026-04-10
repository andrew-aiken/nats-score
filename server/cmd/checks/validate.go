package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"server/pkg/config"
	"server/pkg/nats"

	"github.com/aaiken/nats-score/checks/dns"
	"github.com/aaiken/nats-score/checks/http"
	"github.com/aaiken/nats-score/checks/icmp"
	"github.com/aaiken/nats-score/checks/noop"
	"github.com/aaiken/nats-score/checks/ssh"

	"github.com/creasty/defaults"
	natsnats "github.com/nats-io/nats.go"
)

type Check struct {
	Definition    json.RawMessage `json:"definition"`    // Parameters for the check
	Description   string            `json:"description"`   // Additional information about the check
	Frequency     uint16             `json:"frequency"`     // How often the check runs in seconds
	MutableFields []string          `json:"mutableFields"` // Fields in the definition that can be overwritten
	Name          string            `json:"name"`          // Name of the check
	ScoreWeight   uint8              `json:"scoreWeight"`   // How many points to assign the check
	Type          string            `json:"type"`          // Type of check
}

func Validate(checkName string) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if checkName == "" {
		fmt.Println("Check name required")
		return nil
	}

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config")
		return err
	}

	// Connect to NATS settings KV
	var natsKVClient *nats.NATSKVClient
	natsKVClient, err = nats.NewNATSKVClient(cfg.NATSUrl, cfg.NATSCredsFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client")
		return err
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsKVClient.Close()
	}

	// Setup NATS key value handler
	kv := natsKVClient.GetKVClient()

	checkKey := fmt.Sprintf("check.%s", checkName)

	key, err := kv.Get(checkKey)
	if err == natsnats.ErrKeyNotFound {
		return errors.New("Check does not exist")
	} else if err != nil {
		return err
	}

	data := key.Value()

	var check Check
	if err := json.Unmarshal(data, &check); err != nil {
		return err
	}

	if err := validateCheck(check); err != nil {
		return err
	}

	var checkDefinition any

	// Skip unmarshalling if definition is empty or null
	if len(check.Definition) == 0 || string(check.Definition) == "null" {
		return fmt.Errorf("Definition not defined")
	}

	// Unmarshal Definition based on Type
	switch check.Type {
	case "dns":
		checkDefinition = &dns.Definition{}
	case "http":
		checkDefinition = &http.Definition{}
	case "icmp":
		checkDefinition = &icmp.Definition{}
	case "noop":
		checkDefinition = &noop.Definition{}
	case "ssh":
		checkDefinition = &ssh.Definition{}
	default:
		// For unknown types, keep as check JSON map
		checkDefinition = &noop.Definition{}
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
		fmt.Printf("Check %q validation passed\n", checkName)
	} else {
		fmt.Printf("Check %q validation failed: %s\n", checkName, msg)
	}

	return nil
}

func validateCheck(check Check) error {
	if check.Frequency == 0 {
		fmt.Println("Warning | Frequency undefined, will default to 60 seconds")
	}

	if check.Name == "" {
		fmt.Println("The checks name not defined, should be set to the filename to reduct confusion")
	}

	if check.ScoreWeight == 0 {
		return fmt.Errorf("The checks weight is not defined.")
	}

	if check.Type == "" {
		return fmt.Errorf("Check type not defined")
	}
	return nil
}
