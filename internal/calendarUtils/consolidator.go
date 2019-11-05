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
	endTime := time.Now().AddDate(0, 6, 0).Format(time.RFC3339)

	var inputEvents map[string]*calendar.Event = make(map[string]*calendar.Event)
	var outputEvents map[string]*calendar.Event = make(map[string]*calendar.Event)

	// Get all events part of the Calendar IDs, add them to the inputEvents map
	for _, calendarId :=  range inputCalendarIds {
		events, err := calendarService.Events.List(calendarId).
			SingleEvents(true).TimeMin(t).TimeMax(endTime).Do()
		if err != nil {
			log.Fatalf("ConsolidateCalendar encountered an " +
				"error looking up calendar %v\n", calendarId)
		}
		for _, event := range events.Items {
			if strings.Index(event.Summary, "Birthday") != -1 {
				continue
			} else if strings.Index(event.Summary, "TCF") != -1 {
				continue
			}
			inputEvents[event.Summary + event.Start.DateTime] = event
		}
	}

	outputEventsList, err := calendarService.Events.List(outputCalendarId).
		SingleEvents(true).TimeMin(t).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve output events with calendarID: %v\n", outputCalendarId)
	}

	// convert the outputEvents to a map
	for _, event := range outputEventsList.Items {
		outputEvents[event.Summary + event.Start.DateTime] = event
	}

	// calculate the delta
	eventsToAdd, eventsToRemove := deltaEvents(inputEvents, outputEvents)

	// check each event in the consolidated list and see if it is already added
	log.Printf("Processed %d input events.\n", len(inputEvents))
	log.Printf("Processed %d output events.\n", len(outputEvents))
	log.Printf("Found %d events to Add.\n", len(eventsToAdd))
	log.Printf("Found %d events to Remove.\n", len(eventsToRemove))

	for _, eventToAdd := range eventsToAdd {
		newEvent, _ := rebuildEvent(eventToAdd)
		_, err := calendarService.Events.Insert(outputCalendarId, newEvent).Do()
		if err != nil {
			log.Printf("ERROR: Unable to publish event: %v to CalendarId: %v\n ERROR: %v\n",
				eventToAdd.Summary, outputCalendarId, err)
			log.Printf("ERROR: Start date: %v\n End Date: %v\n", eventToAdd.Start, eventToAdd.End)
			log.Fatalf("ERROR: Object dump %v", eventToAdd)
		} else {
			log.Printf("Published event: %+v\n", eventToAdd.Summary)
		}
	}

	for _, eventToRemove := range eventsToRemove {
		// remove the events
		err := calendarService.Events.Delete(outputCalendarId, eventToRemove.Id).Do()
		if err != nil {
			log.Printf("ERROR: Unable to delete event ID: %v \n Title: %v", eventToRemove.Id, eventToRemove.Summary)
		} else {
			log.Printf("Deleted event: %+v\n", eventToRemove.Summary)
		}
	}

	//for _, inputEvent := range inputEvents {
	//	if _, ok := outputEvents[inputEvent.Summary + inputEvent.Start.DateTime]; !ok {
	//		// doesn't exist, add it to the outputEvents,
	//		// and invoke and add to the Google Calendar
	//
	//		outputEvents[inputEvent.Id] = inputEvent
	//		newEvent, _ := rebuildEvent(inputEvent)
	//		_, err := calendarService.Events.Insert(outputCalendarId, newEvent).Do()
	//		if err != nil {
	//			log.Printf("ERROR: Unable to publish event: %v to CalendarId: %v\n ERROR: %v\n",
	//				inputEvent.Summary, outputCalendarId, err)
	//			log.Printf("ERROR: Start date: %v\n End Date: %v\n", inputEvent.Start, inputEvent.End)
	//			log.Fatalf("ERROR: Object dump %v", inputEvent)
	//		} else {
	//			log.Printf("Published event: %+v\n", inputEvent.Summary)
	//		}
	//	}
	//}

	return nil
}

// Rebuilds an Event to be inserted
// Copies only the: Title, Start Time, End Time, Description
func rebuildEvent(inputEvent *calendar.Event) (*calendar.Event, error) {
	outputEvent := new(calendar.Event)

	outputEvent.Summary = inputEvent.Summary
	outputEvent.Description = inputEvent.Description
	outputEvent.ColorId = inputEvent.ColorId
	outputEvent.Start = inputEvent.Start
	outputEvent.End = inputEvent.End

	return outputEvent, nil
}


func deltaEvents(inputMapping map[string]*calendar.Event, outputMapping map[string]*calendar.Event) (eventsToAdd map[string]*calendar.Event, eventsToRemove map[string]*calendar.Event) {
	eventsToAdd = make(map[string]*calendar.Event)
	eventsToRemove = make(map[string]*calendar.Event)
	for inputKey, inputEvent := range inputMapping {
		if _, ok := outputMapping[inputKey]; !ok {
			eventsToAdd[inputKey] = inputEvent
		}
	}

	for outputKey, outputEvent := range outputMapping {
		if _, ok := inputMapping[outputKey]; !ok {
			eventsToRemove[outputKey] = outputEvent
		}
	}

	return eventsToAdd, eventsToRemove
}