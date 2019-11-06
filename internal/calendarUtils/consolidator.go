package calendarUtils

import (
	"google.golang.org/api/calendar/v3"
	"log"
	"time"
)

const CLEANUP = false

type CalendarMap map[string]*calendar.Event

// Directional Consolidation of Calendar events
func ConsolidateCalendars(inputCalendarIds []string,
	outputCalendarId string, calendarService *calendar.Service) error {

	var inputEvents CalendarMap = make(CalendarMap)
	var outputEvents CalendarMap = make(CalendarMap)
	//var keepEvents CalendarMap = make(CalendarMap)
	var eventsToAdd CalendarMap = make(CalendarMap)
	var eventsToRemove CalendarMap = make(CalendarMap)

	// build the start and end times
	currentTime := time.Now()

	startScanTime := currentTime.AddDate(0, 0, -7).Format(time.RFC3339)
	endScanTime := currentTime.AddDate(0, 1, 0).Format(time.RFC3339)
	//startKeepTime := currentTime.AddDate(0, -1, 0).Format(time.RFC3339)
	//endKeepTime := currentTime.AddDate(0, -1, 0).Format(time.RFC3339)

	// Grab all the input events in the sliding window
	for _, calendarId := range inputCalendarIds {
		err := getAllEvents(calendarId, startScanTime, endScanTime, &inputEvents, calendarService)
		if err != nil {
			log.Fatalf("Error: Unable to retrieve events for calendar: %v", calendarId)
		}
	}

	// Grab all the output events in the sliding window
	err := getAllEvents(outputCalendarId, startScanTime, endScanTime, &outputEvents, calendarService)
	if err != nil {
		log.Fatalf("Error: Unable to retrieve events for calendar: %v", outputCalendarId)
	}

	log.Printf("Processed %d input events\n", len(inputEvents))
	log.Printf("Processed %d output events\n", len(outputEvents))

	// Calculate the delta
	calculateDelta(inputEvents, outputEvents, &eventsToAdd, &eventsToRemove)

	log.Printf("Found %d events to add\n", len(eventsToAdd))
	log.Printf("Found %d events to remove\n", len(eventsToRemove))

	// Perform the changes
	performChanges(outputCalendarId, eventsToAdd, eventsToRemove, calendarService)

	return nil
}

func performChanges(calendarId string, eventsToAdd CalendarMap,
	eventsToRemove CalendarMap, service *calendar.Service) {

	for _, eventToAdd := range eventsToAdd {
		newEvent, _ := rebuildEvent(eventToAdd)
		_, err := service.Events.Insert(calendarId, newEvent).Do()
		if err != nil {
			log.Printf("ERROR: Unable to publish event: %v to CalendarId: %v\n ERROR: %v\n",
				eventToAdd.Summary, calendarId, err)
			log.Printf("ERROR: Start date: %v\n End Date: %v\n", eventToAdd.Start, eventToAdd.End)
			log.Fatalf("ERROR: Object dump %v", eventToAdd)
		} else {
			log.Printf("Published event: %+v\n", eventToAdd.Summary)
		}
	}

	for _, eventToRemove := range eventsToRemove {
		// remove the events
		err := service.Events.Delete(calendarId, eventToRemove.Id).Do()
		if err != nil {
			log.Printf("ERROR: Unable to delete event ID: %v \n Title: %v", eventToRemove.Id, eventToRemove.Summary)
		} else {
			log.Printf("Deleted event: %+v\n", eventToRemove.Summary)
		}
	}
}

func calculateDelta(eventsToSync CalendarMap, currentEvents CalendarMap,
	eventsToAddRef *CalendarMap, eventsToRemoveRef *CalendarMap) {

	eventsToAdd := *eventsToAddRef
	eventsToRemove := *eventsToRemoveRef

	for eventKey, event := range eventsToSync {
		if _, ok := currentEvents[eventKey]; !ok {
			eventsToAdd[eventKey] = event
		}
	}

	for eventKey, event := range currentEvents {
		if _, ok := eventsToSync[eventKey]; !ok {
			eventsToRemove[eventKey] = event
		}
	}
}

func getAllEvents(calendarId string, startDate string, endDate string,
	resultantEventsRef *CalendarMap, service *calendar.Service) error {
	resultantEvents := *resultantEventsRef
	events, err := service.Events.List(calendarId).
		TimeMin(startDate).TimeMax(endDate).
		SingleEvents(true).Do()
	if err != nil {
		log.Printf("Error: Unable to retrieve events using ID: %v\n", calendarId)
		return err
	}

	for _, event := range events.Items {
		//if event == nil {
		//	continue
		//}
		eventKey := generateEventMapKey(event)
		//eventKey := event.Summary + " : " + event.Start.DateTime + " : " + event.End.DateTime
		if _, ok := resultantEvents[eventKey]; !ok {
			resultantEvents[eventKey] = event
		} else {
			log.Printf("Duplicate event found with key: %v\n", eventKey)
			if CLEANUP {
				_ = service.Events.Delete(calendarId, event.Id).Do()
			}
		}
	}

	return nil
}
/*
func ConsolidateCalendars_Old(inputCalendarIds []string,
	outputCalendarId string, calendarService *calendar.Service) error {

	startDate := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	endTime := time.Now().AddDate(0, 2, 0).Format(time.RFC3339)
	minKeepDate := time.Now().AddDate(0, -2, 0).Format(time.RFC3339)
	maxKeepDate := time.Now().AddDate(0, 2, 0).Format(time.RFC3339)

	var inputEvents map[string]*calendar.Event = make(map[string]*calendar.Event)
	var outputEvents map[string]*calendar.Event = make(map[string]*calendar.Event)
	var keepEvents map[string]*calendar.Event = make(map[string]*calendar.Event)
	var eventsToAdd map[string]*calendar.Event = make(map[string]*calendar.Event)
	var eventsToRemove map[string]*calendar.Event = make(map[string]*calendar.Event)

	// Get all events part of the Calendar IDs, add them to the inputEvents map
	for _, calendarId := range inputCalendarIds {
		events, err := calendarService.Events.List(calendarId).
			SingleEvents(true).TimeMin(startDate).TimeMax(endTime).Do()
		if err != nil {
			log.Fatalf("ConsolidateCalendar encountered an "+
				"error looking up calendar %v\n", calendarId)
		}
		for _, event := range events.Items {
			var eventKey string = generateEventMapKey(event)
			if strings.Index(event.Summary, "Birthday") != -1 {
				continue
			} else if strings.Index(event.Summary, "TCF") != -1 {
				continue
			}
			if _, ok := inputEvents[eventKey]; ok {

			} else {
				inputEvents[eventKey] = event
			}
		}

		eventsToKeep, err := calendarService.Events.List(calendarId).
			SingleEvents(true).TimeMin(minKeepDate).TimeMax(maxKeepDate).Do()
		if err != nil {
			log.Fatalf("ConsolidateCalendar encountered an "+
				"error looking up calendar %v\n", calendarId)
		}
		for _, event := range eventsToKeep.Items {
			var eventKey string = generateEventMapKey(event)

			keepEvents[eventKey] = event
		}
	}

	outputEventsList, err := calendarService.Events.List(outputCalendarId).
		SingleEvents(true).TimeMin(startDate).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve output events with calendarID: %v\n", outputCalendarId)
	}

	// convert the outputEvents to a map
	for _, event := range outputEventsList.Items {
		var eventKey string = generateEventMapKey(event)

		if _, ok := outputEvents[eventKey]; ok {
			log.Printf("Found duplicate entry with name: %v and time: %v\n", event.Summary, event.Start.DateTime)
			eventsToRemove[event.Id] = event
		} else {
			outputEvents[eventKey] = event
		}
	}

	// calculate the delta
	deltaEvents(inputEvents, outputEvents, keepEvents, &eventsToAdd, &eventsToRemove)

	// check each event in the consolidated list and see if it is already added
	log.Printf("Processed %d input events.\n", len(inputEvents))
	log.Printf("Processed %d output events.\n", len(outputEvents))
	log.Printf("Processed %d events to keep.\n", len(keepEvents))
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

	return nil
}
*/

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

/*
func deltaEvents(
	newEvents map[string]*calendar.Event,
	currentEvents map[string]*calendar.Event,
	keepMapping map[string]*calendar.Event,
	eventsToAddRef *map[string]*calendar.Event,
	eventsToRemoveRef *map[string]*calendar.Event,
) {
	//eventsToAdd = make(map[string]*calendar.Event)
	//eventsToRemove = make(map[string]*calendar.Event)

	eventsToAdd := *eventsToAddRef
	eventsToRemove := *eventsToRemoveRef

	for newEventKey, newEvent := range newEvents {
		if _, ok := currentEvents[newEventKey]; !ok {
			log.Printf("Planning to add event: %v at: %v\n", newEvent.Summary, newEvent.Start.DateTime)
			eventsToAdd[newEventKey] = newEvent
		}
	}

	for outputKey, outputEvent := range currentEvents {
		if _, ok := keepMapping[outputKey]; !ok {
			eventsToRemove[outputKey] = outputEvent
		}
	}
}
*/

func generateEventMapKey(event *calendar.Event) string {
	if event == nil {
		log.Fatalf("Error: Unable to access event due to invalid memory address: %v", event)
	}

	//log.Printf("Event%v\n", event)
	//log.Printf("Event: %v with %v\n", event.Summary, event.OriginalStartTime.DateTime)
	return event.Summary + " : " + event.Start.DateTime + " : "+ event.End.DateTime
}
