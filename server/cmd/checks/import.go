package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"server/internal/config"
	"server/internal/nats"
)

func Import(directory string) error {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config")
		return err
	}

	// Validate check directory
	if err := validateDirectory(directory); err != nil {
		return err
	}

	// Connect to NATS settings KV
	natsClient := nats.NatsConnection{
		NatsUrl:       cfg.NATSUrl,
		NatsCredsFile: cfg.NATSCredsFile,
	}
	err = natsClient.SetupConnection()
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS KV client")
		return err
	} else {
		log.Println("Connected to NATS KV bucket 'settings'")
		defer natsClient.Close()
	}

	// Read files in directory
	checkFiles, err := readDirectory(directory)
	if err != nil {
		return err
	}

	err = loadKV(directory, checkFiles, natsClient)

	return err
}

func validateDirectory(directoryPath string) error {
	info, err := os.Stat(directoryPath)

	if err == nil && info.IsDir() {
		return nil
	}

	return err
}

func readDirectory(directoryPath string) ([]string, error) {
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}

	var validFiles = []string{}

	for _, file := range entries {
		fileName := file.Name()
		if strings.HasSuffix(fileName, ".json") {
			validFiles = append(validFiles, fileName)
		}
	}

	return validFiles, nil
}

func loadKV(directory string, checkFiles []string, natsClient nats.NatsConnection) error {
	for _, file := range checkFiles {
		checkName := fmt.Sprintf("check.%s", strings.TrimSuffix(file, ".json"))

		filePath := fmt.Sprintf("%s/%s", directory, file)

		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("failed reading file: %s", err)
		}

		dst := &bytes.Buffer{}

		if err := json.Compact(dst, []byte(data)); err != nil {
			return err
		}

		natsClient.NatsKV.Put(checkName, dst.Bytes())
	}

	log.Println("Checks written to KV")

	return nil
}
