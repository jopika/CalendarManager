package calendarUtils

import (
	"../configManager"
	"google.golang.org/api/calendar/v3"
	"log"
	"strings"
	"time"
)

const CLEANUP = false
const DELETE_ALL = false

type CalendarMap map[string]*calendar.Event

// Directional Consolidation of Calendar events
func ConsolidateCalendars(inputCalendarIds []string, outputCalendarId string,
	config configManager.Config, calendarService *calendar.Service) error {

	var inputEvents CalendarMap = make(CalendarMap)
	var outputEvents CalendarMap = make(CalendarMap)
	var eventsToAdd CalendarMap = make(CalendarMap)
	var eventsToRemove CalendarMap = make(CalendarMap)

	// build the start and end times
	currentTime := time.Now()

	startScanTime := currentTime.AddDate(0, 0, -7).Format(time.RFC3339)
	endScanTime := currentTime.AddDate(0, 2, 0).Format(time.RFC3339)

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
	performChanges(outputCalendarId, eventsToAdd, eventsToRemove, calendarService, config.BlacklistedWords)

	return nil
}

func eventDecision(event *calendar.Event, blacklistedWords []string) bool {
	for _, word := range blacklistedWords {
		if strings.Index(event.Summary, word) != -1 {
			log.Printf("Skipping event: %v\n", event.Summary)
			return false
		}
	}

	return true
}

func performChanges(calendarId string, eventsToAdd CalendarMap,
	eventsToRemove CalendarMap, service *calendar.Service, blacklistedWords []string) {

	for _, eventToAdd := range eventsToAdd {
		if !eventDecision(eventToAdd, blacklistedWords) {
			continue
		}
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
		if DELETE_ALL {
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

// Rebuilds an Event to be inserted
// Copies only the: Title, Start Time, End Time, Description
func rebuildEvent(inputEvent *calendar.Event) (*calendar.Event, error) {
	outputEvent := new(calendar.Event)

	outputEvent.Summary = inputEvent.Summary
	outputEvent.Description = inputEvent.Description
	outputEvent.ColorId = inputEvent.ColorId
	outputEvent.Location = inputEvent.Location
	outputEvent.Start = inputEvent.Start
	outputEvent.End = inputEvent.End

	return outputEvent, nil
}

func generateEventMapKey(event *calendar.Event) string {
	if event == nil {
		log.Fatalf("Error: Unable to access event due to invalid memory address: %v", event)
	}
	return event.Summary + " : " + event.Start.DateTime + " : "+ event.End.DateTime
}
