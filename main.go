package main

import (
	"./internal/calendarUtils"
	"./internal/configManager"
	"context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"time"
)

func main() {
	srv := buildClient()

	config := configManager.LoadConfiguration("./config.json")

	var outputCalendarId string
	var inputCalendarIds []string

	inputCalendarIds, outputCalendarId = retrieveCalendarIds(config)

	var complete chan int = make(chan int)

	go func() {
		for {
			log.Println("Beginning to consolidate calendar")
			err := calendarUtils.ConsolidateCalendars(inputCalendarIds, outputCalendarId, config, srv)
			if err != nil {
				log.Fatalf("Unable to consolidate calendars\n")
			}

			log.Println("Sleeping...")
			// sleep for a bit
			time.Sleep(time.Duration(config.SyncIntervalMins) * time.Minute)
		}
	}()

	<-complete
}

// Builds the client and returns it
func buildClient() *calendar.Service {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope, calendar.CalendarEventsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to configManager: %v", err)
	}
	client := calendarUtils.GetClient(config)
	ctx := context.Background()

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	return srv
}

func retrieveCalendarIds(config configManager.Config) (inputCalendarIds []string, outputCalendarId string) {
	switch config.Environment {
	case configManager.Production:
		inputCalendarIds = config.InputCalendarIds
		outputCalendarId = config.OutputCalendarId
		break
	case configManager.Test:
	case configManager.Dev:
		inputCalendarIds = config.TestInputCalendarIds
		outputCalendarId = config.TestOutputCalendarId
	}

	return inputCalendarIds, outputCalendarId
}
