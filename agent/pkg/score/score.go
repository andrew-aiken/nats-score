package score

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/aaiken/nats-score/pkg/checks"
	"github.com/aaiken/nats-score/pkg/config"
	"github.com/nats-io/nats.go"
)

func HandleScoreEvent(settings *config.Settings, js nats.JetStreamContext) nats.MsgHandler {
	return func(msg *nats.Msg) {
		checkName := strings.TrimPrefix(msg.Subject, "events.score.")

		// fmt.Println(settings.Attributes)

		streamName := "results." + strconv.Itoa(settings.StaticConf.TeamNumber) + "." + checkName

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

		// Override the mutable fields with the team setting attributes
		var override map[string]string
		allowedArgumentOverrides(value.MutableFields, settings.Attributes[checkName], &override)

		// Apply overrides to the check definition
		if err := applyOverrides(value.Definition, override); err != nil {
			log.Printf("Failed to apply overrides for check %s: %v", checkName, err)
		}

		ctx := context.Background()
		result := checker.Run(ctx, input, settings.StaticConf)

		if err := publishResults(streamName, result, value.ScoreWeight, js); err != nil {
			log.Printf("Failed to publish results: %v", err)
		}

		log.Printf("Check %s result: %v", checkName, result.Passed)
		for key, value := range result.Details {
			log.Printf("Check %s detail: %s = %s", checkName, key, value)
		}
	}
}

func allowedArgumentOverrides(allowedItems []string, attributes map[string]string, override *map[string]string) {
	*override = make(map[string]string)
	for key, value := range attributes {
		for _, allowed := range allowedItems {
			if key == allowed {
				(*override)[key] = value
			}
		}
	}
}

// TODO: This code stuff can probably be refactored
// applyOverrides uses reflection to set field values on the definition struct
func applyOverrides(definition interface{}, overrides map[string]string) error {
	if len(overrides) == 0 {
		return nil
	}

	val := reflect.ValueOf(definition)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("definition must be a non-nil pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("definition must point to a struct")
	}

	typ := val.Type()

	for key, value := range overrides {
		// Find the field by name (case-insensitive match)
		var field reflect.Value
		var found bool
		for i := 0; i < typ.NumField(); i++ {
			if strings.EqualFold(typ.Field(i).Name, key) {
				field = val.Field(i)
				found = true
				break
			}
		}

		if !found {
			log.Printf("Field %s not found in definition", key)
			continue
		}

		if !field.CanSet() {
			log.Printf("Field %s cannot be set", key)
			continue
		}

		// Convert string value to appropriate type
		switch field.Kind() {
		case reflect.String:
			field.SetString(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int value for field %s: %w", key, err)
			}
			field.SetInt(intVal)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("failed to parse bool value for field %s: %w", key, err)
			}
			field.SetBool(boolVal)
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("failed to parse float value for field %s: %w", key, err)
			}
			field.SetFloat(floatVal)
		default:
			return fmt.Errorf("unsupported field type %s for field %s", field.Kind(), key)
		}
	}

	return nil
}
