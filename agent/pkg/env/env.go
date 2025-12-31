package env

import (
	"fmt"
	"os"
	"strconv"
)

type InputVariables struct {
	NatsUrl       string
	NatsCredsFile string
	TeamNumberInt int
	// Definitely a cleaner way to do this
	TeamNumberString string
}

// ValidateEnvironmentVariables reads environment variables and returns required startup arguments
func ValidateEnvironmentVariables() (InputVariables, error) {

	var inputs InputVariables

	inputs.NatsUrl = os.Getenv("NATS_URL")
	if inputs.NatsUrl == "" {
		fmt.Println("No nats url specified using default address")
		inputs.NatsUrl = "nats://127.0.0.1:4222"
	}

	inputs.NatsCredsFile = os.Getenv("NATS_CREDS_FILE")
	if inputs.NatsCredsFile == "" {
		return inputs, fmt.Errorf("No credentials file specified in env variable 'NATS_CREDS_FILE'")
	}

	inputs.TeamNumberString = os.Getenv("TEAM_NUMBER")
	if inputs.TeamNumberString == "" {
		return inputs, fmt.Errorf("No team number specified in env variable 'TEAM_NUMBER'")
	}

	var err error

	inputs.TeamNumberInt, err = strconv.Atoi(inputs.TeamNumberString)
	if err != nil {
		return inputs, fmt.Errorf("Invalid TEAM_NUMBER (%v) Must be an integer", err)
	}

	return inputs, nil
}
