package calendarUtils

import (
	"google.golang.org/api/calendar/v3"
	"log"
	"strings"
	"time"
)

func ConsolidateCalendars(inputCalendarIds []string,
	outputCalendarId string, calendarService *calendar.Service) error {
	t := time.Now().Format(time.RFC3339)
	var consolidatedEventsList []*calendar.Event = make([]*calendar.Event, 0)
	// Get all events part of the Calendar IDs
	for _, calendarId :=  range inputCalendarIds {
		events, err := calendarService.Events.List(calendarId).
			SingleEvents(true).TimeMin(t).Do()
		if err != nil {
			log.Fatalf("ConsolidateCalendar encountered an " +
				"error looking up calendar %v\n", calendarId)
		}
		for _, event := range events.Items {
			consolidatedEventsList = append(consolidatedEventsList, event)
		}
	}

	outputEventsList, err := calendarService.Events.List(outputCalendarId).
		SingleEvents(true).TimeMin(t).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve output events with calendarID: %v\n", outputCalendarId)
	}

	// convert the outputEvents to a map
	var outputEvents map[string]*calendar.Event = make(map[string]*calendar.Event)

	for _, event := range outputEventsList.Items {
		outputEvents[event.Id] = event
	}

	// check each event in the consolidated list and see if it is already added
	log.Printf("Found %d events.\n", len(consolidatedEventsList))
	for _, inputEvent := range consolidatedEventsList {
		if strings.Index(inputEvent.Summary, "Birthday") != -1 {
			continue;
		} else if strings.Index(inputEvent.Summary, "TCF") != -1 {
			continue;
		}
		if _, ok := outputEvents[inputEvent.Id]; !ok {
			// doesn't exist, add it to the outputEvents,
			// and invoke and add to the Google Calendar

			outputEvents[inputEvent.Id] = inputEvent
			_, err := calendarService.Events.Insert(outputCalendarId, inputEvent).Do()
			if err != nil {
				log.Printf("ERROR: Unable to publish event: %v to CalendarId: %v\n ERROR: %v\n",
					inputEvent.Summary, outputCalendarId, err)
			} else {
				log.Printf("Published event: %v", inputEvent.Summary)
			}
		}
	}

	return nil;
}