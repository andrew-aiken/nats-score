package score

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aaiken/nats-score/pkg/checks"
	"github.com/aaiken/nats-score/pkg/config"
	"github.com/nats-io/nats.go"
)

func HandleScoreEvent(settings *config.Settings, js nats.JetStreamContext) nats.MsgHandler {
	return func(msg *nats.Msg) {
		// log.Printf("📢 Ephemeral event received on [%s]: %s", msg.Subject, string(msg.Data))
		// out, err := yaml.Marshal(settings)
		// if err != nil {
		// 	log.Printf("Failed to marshal settings to YAML: %v", err)
		// } else {
		// 	fmt.Println(string(out))
		// }

		checkName := strings.TrimPrefix(msg.Subject, "events.score.")

		streamName := "results." + strconv.Itoa(settings.TeamNumber) + "." + checkName

		var input map[string]interface{}

		if string(msg.Data) != "" {
			if err := json.Unmarshal(msg.Data, &input); err != nil {
				log.Printf("Failed to unmarshal check arguments: %v", err)
				return
			}
		}

		value, ok := settings.Checks[checkName]

		if !ok {
			fmt.Printf("Check %s not found\n", checkName)
			return
		}

		// Type assert the definition to the Checker interface and call Run
		checker, ok := value.Definition.(checks.Checker)
		if !ok {
			fmt.Printf("Check %s definition does not implement Checker interface\n", checkName)
			return
		}

		ctx := context.Background()
		result := checker.Run(ctx, input)

		if err := publishResults(streamName, result, value.ScoreWeight, js); err != nil {
			log.Printf("Failed to publish results: %v", err)
		}

		log.Printf("Check %s result: %v error: %s", checkName, result.Passed, result.Message)
		for key, value := range result.Details {
			log.Printf("Check %s detail: %s = %s", checkName, key, value)
		}
	}
}
