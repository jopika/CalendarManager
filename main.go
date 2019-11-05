package main

import (
	"./internal/calendarUtils"
	"context"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"time"
)

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope, calendar.CalendarEventsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := calendarUtils.GetClient(config)
	ctx := context.Background()

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}
	fmt.Println("Upcoming events:")
	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}
			fmt.Printf("%v (%v)\n", item.Summary, date)
		}
	}

	calendars, err := srv.CalendarList.List().Do()
	if len(calendars.Items) == 0 {
		fmt.Println("No calendars found through the API.")
	} else {
		for _, calendar := range calendars.Items {
			fmt.Printf("Calendar found: %v with description: %v\n", calendar.Id, calendar.Summary)
		}
	}

	// verify we have write access to the output calendar
	var outputCalendarId string = "4soqi2o4i24d1lpal0lkc6iuhg@group.calendar.google.com"
	var inputCalendarIds []string = []string{"primary", "n505ujqlrdsec5t50vtur8tub8@group.calendar.google.com"}

	//var inputCalendarIds []string = []string{"primary"}

	outputCalendarInfo, err := srv.CalendarList.Get(outputCalendarId).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve information for output calendar\n")
	}

	log.Printf("Output calendar Info: %v\n", outputCalendarInfo)
	log.Printf("Access Role: %v\n", outputCalendarInfo.AccessRole)

	err = calendarUtils.ConsolidateCalendars(inputCalendarIds, outputCalendarId, srv)
	if err != nil {
		log.Fatalf("Unable to consolidate calendar\n")
	}
}
