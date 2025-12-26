package caldav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/teambition/rrule-go"
)

type Event struct {
	UID          string
	Summary      string
	Description  string
	Location     string
	Start        time.Time
	End          time.Time
	RRULE        string
	EXDATEs      []time.Time
	RecurrenceID *time.Time
	CalendarPath string
	CalendarName string
}

func (c *Client) ListEvents(ctx context.Context, calendarPath string, start, end time.Time) ([]Event, error) {
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name:  "VCALENDAR",
			Props: []string{"VERSION"},
			Comps: []caldav.CalendarCompRequest{
				{
					Name: "VEVENT",
					Props: []string{
						"UID",
						"SUMMARY",
						"DESCRIPTION",
						"LOCATION",
						"DTSTART",
						"DTEND",
						"RRULE",
						"EXDATE",
						"DURATION",
					},
				},
			},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{
				{
					Name:  "VEVENT",
					Start: start,
					End:   end,
				},
			},
		},
	}

	objects, err := c.caldavClient.QueryCalendar(ctx, calendarPath, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar: %w", err)
	}

	var events []Event
	for _, obj := range objects {
		if obj.Data == nil {
			continue
		}
		for _, comp := range obj.Data.Children {
			if comp.Name != "VEVENT" {
				continue
			}
			expanded := expandEvent(comp, start, end)
			events = append(events, expanded...)
		}
	}

	return events, nil
}

func expandEvent(comp *ical.Component, rangeStart, rangeEnd time.Time) []Event {
	var uid, summary, description, location, rruleStr string
	var eventStart, eventEnd time.Time
	var duration time.Duration
	var exdates []time.Time

	if props := comp.Props.Get(ical.PropUID); props != nil {
		uid = props.Value
	}
	if props := comp.Props.Get(ical.PropSummary); props != nil {
		summary = props.Value
	}
	if props := comp.Props.Get(ical.PropDescription); props != nil {
		description = props.Value
	}
	if props := comp.Props.Get(ical.PropLocation); props != nil {
		location = props.Value
	}
	if props := comp.Props.Get(ical.PropRecurrenceRule); props != nil {
		rruleStr = props.Value
	}
	for _, prop := range comp.Props.Values(ical.PropExceptionDates) {
		exdates = append(exdates, parseICalTime(prop))
	}
	if props := comp.Props.Get(ical.PropDateTimeStart); props != nil {
		eventStart = parseICalTime(*props)
	}
	if props := comp.Props.Get(ical.PropDateTimeEnd); props != nil {
		eventEnd = parseICalTime(*props)
	}
	if !eventEnd.IsZero() && !eventStart.IsZero() {
		duration = eventEnd.Sub(eventStart)
	}
	if props := comp.Props.Get(ical.PropDuration); props != nil {
		if d, err := time.ParseDuration(strings.ReplaceAll(strings.ToLower(props.Value), "pt", "") + "s"); err == nil {
			duration = d
		}
	}

	rruleOpt, err := comp.Props.RecurrenceRule()
	if err != nil || rruleOpt == nil {
		if !eventStart.Before(rangeStart) && eventStart.Before(rangeEnd) {
			return []Event{{
				UID:         uid,
				Summary:     summary,
				Description: description,
				Location:    location,
				Start:       eventStart,
				End:         eventEnd,
				RRULE:       rruleStr,
				EXDATEs:     exdates,
			}}
		}
		return nil
	}

	rruleOpt.Dtstart = eventStart
	rule, err := rrule.NewRRule(*rruleOpt)
	if err != nil {
		return []Event{{
			UID:         uid,
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       eventStart,
			End:         eventEnd,
			RRULE:       rruleStr,
			EXDATEs:     exdates,
		}}
	}

	occurrences := rule.Between(rangeStart, rangeEnd, true)
	var events []Event
	for _, occStart := range occurrences {
		if isExcluded(occStart, exdates) {
			continue
		}
		events = append(events, Event{
			UID:         uid,
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       occStart,
			End:         occStart.Add(duration),
			RRULE:       rruleStr,
			EXDATEs:     exdates,
		})
	}

	return events
}

func isExcluded(t time.Time, exdates []time.Time) bool {
	for _, exdate := range exdates {
		if t.Year() == exdate.Year() && t.Month() == exdate.Month() && t.Day() == exdate.Day() &&
			t.Hour() == exdate.Hour() && t.Minute() == exdate.Minute() {
			return true
		}
	}
	return false
}

func (c *Client) ListEventsAllCalendars(ctx context.Context, start, end time.Time) ([]Event, error) {
	calendars, err := c.ListCalendars(ctx)
	if err != nil {
		return nil, err
	}

	var allEvents []Event
	for _, cal := range calendars {
		if !cal.SupportsEvents() {
			continue
		}
		events, err := c.ListEvents(ctx, cal.Path, start, end)
		if err != nil {
			continue
		}
		for i := range events {
			events[i].CalendarPath = cal.Path
			events[i].CalendarName = cal.Name
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (c *Client) CreateEvent(ctx context.Context, calendarPath string, event Event) (*Event, error) {
	if event.UID == "" {
		event.UID = uuid.New().String()
	}

	eventPath := path.Join(calendarPath, event.UID+".ics")
	eventURL := iCloudCalDAVURL + eventPath

	icsData := buildICS(event)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, eventURL, bytes.NewBufferString(icsData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	req.Header.Set("If-None-Match", "*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create event: %s - %s", resp.Status, string(respBody))
	}

	return &event, nil
}

func (c *Client) UpdateEvent(ctx context.Context, calendarPath string, event Event) error {
	if event.UID == "" {
		return fmt.Errorf("event UID is required for update")
	}

	eventPath := path.Join(calendarPath, event.UID+".ics")
	eventURL := iCloudCalDAVURL + eventPath

	icsData := buildICS(event)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, eventURL, bytes.NewBufferString(icsData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update event: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) DeleteEvent(ctx context.Context, calendarPath, eventUID string) error {
	eventPath := path.Join(calendarPath, eventUID+".ics")
	eventURL := iCloudCalDAVURL + eventPath

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, eventURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete event: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) DeleteEventOccurrence(ctx context.Context, calendarPath, eventUID string, occurrenceTime time.Time) error {
	event, err := c.GetEvent(ctx, calendarPath, eventUID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if event.RRULE == "" {
		return fmt.Errorf("event is not recurring; use regular delete instead")
	}

	event.EXDATEs = append(event.EXDATEs, occurrenceTime)

	if err := c.UpdateEvent(ctx, calendarPath, *event); err != nil {
		return fmt.Errorf("failed to exclude occurrence: %w", err)
	}

	return nil
}

func (c *Client) UpdateEventOccurrence(ctx context.Context, calendarPath, eventUID string, occurrenceTime time.Time, updates Event) error {
	master, err := c.GetEvent(ctx, calendarPath, eventUID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if master.RRULE == "" {
		return fmt.Errorf("event is not recurring; use regular update instead")
	}

	exception := Event{
		UID:          master.UID,
		Summary:      master.Summary,
		Description:  master.Description,
		Location:     master.Location,
		Start:        occurrenceTime,
		End:          occurrenceTime.Add(master.End.Sub(master.Start)),
		RecurrenceID: &occurrenceTime,
	}

	if updates.Summary != "" {
		exception.Summary = updates.Summary
	}
	if updates.Description != "" {
		exception.Description = updates.Description
	}
	if updates.Location != "" {
		exception.Location = updates.Location
	}
	if !updates.Start.IsZero() {
		exception.Start = updates.Start
	}
	if !updates.End.IsZero() {
		exception.End = updates.End
	}

	eventPath := path.Join(calendarPath, eventUID+".ics")
	eventURL := iCloudCalDAVURL + eventPath

	icsData := buildICSWithException(*master, exception)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, eventURL, bytes.NewBufferString(icsData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update occurrence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update occurrence: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) GetEvent(ctx context.Context, calendarPath, eventUID string) (*Event, error) {
	eventPath := path.Join(calendarPath, eventUID+".ics")

	obj, err := c.caldavClient.GetCalendarObject(ctx, eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if obj.Data == nil {
		return nil, fmt.Errorf("event not found")
	}

	for _, comp := range obj.Data.Children {
		if comp.Name != "VEVENT" {
			continue
		}
		events := expandEvent(comp, time.Time{}, time.Now().AddDate(100, 0, 0))
		if len(events) > 0 {
			events[0].CalendarPath = calendarPath
			return &events[0], nil
		}
	}

	return nil, fmt.Errorf("event not found")
}
