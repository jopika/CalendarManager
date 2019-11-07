package configManager

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	InputCalendarIds []string `json:"input_calendar_ids"`
	OutputCalendarId string   `json:"output_calendar_id"`
	SyncIntervalMins int      `json:"sync_interval_mins"`
	TestInputCalendarIds []string `json:"test_input_calendar_ids"`
	TestOutputCalendarId string `json:"test_output_calendar_id"`
	Environment Environment `json:"environment"`
}

type Environment string

const (
	Production = "prod"
	Test = "test"
	Dev = "dev"
)


func LoadConfiguration(filePath string) Config {
	var config Config

	configFile, err := os.Open(filePath)
	defer configFile.Close()
	if err != nil {
		log.Fatalf("Unable to read configManager file.\n Error: %v\n", err)
	}

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		log.Fatalf("Unable to read configManager file.\n Error: %v\n", err)
	}

	return config
}
