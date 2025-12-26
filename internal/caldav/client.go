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
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/teambition/rrule-go"
)

const (
	iCloudCalDAVURL = "https://caldav.icloud.com"
)

type Client struct {
	caldavClient *caldav.Client
	httpClient   webdav.HTTPClient
	email        string
	homeSet      string
}

func NewClient(email, password string) (*Client, error) {
	httpClient := webdav.HTTPClientWithBasicAuth(&http.Client{}, email, password)

	caldavClient, err := caldav.NewClient(httpClient, iCloudCalDAVURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	return &Client{
		caldavClient: caldavClient,
		httpClient:   httpClient,
		email:        email,
	}, nil
}

func (c *Client) FindCalendarHomeSet(ctx context.Context) (string, error) {
	if c.homeSet != "" {
		return c.homeSet, nil
	}

	principal, err := c.caldavClient.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to find current user principal: %w", err)
	}

	homeSet, err := c.caldavClient.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return "", fmt.Errorf("failed to find calendar home set: %w", err)
	}

	c.homeSet = homeSet
	return homeSet, nil
}

type Calendar struct {
	Path                  string
	Name                  string
	Description           string
	Color                 string
	SupportedComponentSet []string
}

func (c *Client) ListCalendars(ctx context.Context) ([]Calendar, error) {
	homeSet, err := c.FindCalendarHomeSet(ctx)
	if err != nil {
		return nil, err
	}

	calendars, err := c.caldavClient.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	result := make([]Calendar, 0, len(calendars))
	for _, cal := range calendars {
		result = append(result, Calendar{
			Path:                  cal.Path,
			Name:                  cal.Name,
			Description:           cal.Description,
			SupportedComponentSet: cal.SupportedComponentSet,
		})
	}

	return result, nil
}

func (c *Client) GetCalendar(ctx context.Context, calendarID string) (*Calendar, error) {
	calendars, err := c.ListCalendars(ctx)
	if err != nil {
		return nil, err
	}

	for _, cal := range calendars {
		if cal.Path == calendarID || strings.HasSuffix(cal.Path, "/"+calendarID+"/") {
			return &cal, nil
		}
	}

	return nil, fmt.Errorf("calendar not found: %s", calendarID)
}

func (c *Client) CreateCalendar(ctx context.Context, name string) (*Calendar, error) {
	homeSet, err := c.FindCalendarHomeSet(ctx)
	if err != nil {
		return nil, err
	}

	calendarID := uuid.New().String()
	calendarPath := path.Join(homeSet, calendarID) + "/"
	calendarURL := iCloudCalDAVURL + calendarPath

	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<mkcalendar xmlns="urn:ietf:params:xml:ns:caldav" xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>%s</D:displayname>
    </D:prop>
  </D:set>
</mkcalendar>`, escapeXML(name))

	req, err := http.NewRequestWithContext(ctx, "MKCALENDAR", calendarURL, bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create calendar: %s - %s", resp.Status, string(respBody))
	}

	return &Calendar{
		Path: calendarPath,
		Name: name,
	}, nil
}

func (c *Client) DeleteCalendar(ctx context.Context, calendarID string) error {
	calendarPath := calendarID
	if !strings.HasPrefix(calendarPath, "/") {
		homeSet, err := c.FindCalendarHomeSet(ctx)
		if err != nil {
			return err
		}
		calendarPath = path.Join(homeSet, calendarID) + "/"
	}

	calendarURL := iCloudCalDAVURL + calendarPath

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, calendarURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete calendar: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func (c *Client) UpdateCalendar(ctx context.Context, calendarID, name string) error {
	calendarPath := calendarID
	if !strings.HasPrefix(calendarPath, "/") {
		homeSet, err := c.FindCalendarHomeSet(ctx)
		if err != nil {
			return err
		}
		calendarPath = path.Join(homeSet, calendarID) + "/"
	}

	calendarURL := iCloudCalDAVURL + calendarPath

	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<D:propertyupdate xmlns:D="DAV:">
  <D:set>
    <D:prop>
      <D:displayname>%s</D:displayname>
    </D:prop>
  </D:set>
</D:propertyupdate>`, escapeXML(name))

	req, err := http.NewRequestWithContext(ctx, "PROPPATCH", calendarURL, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update calendar: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

type Event struct {
	UID          string
	Summary      string
	Description  string
	Location     string
	Start        time.Time
	End          time.Time
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
	var uid, summary, description, location string
	var eventStart, eventEnd time.Time
	var duration time.Duration

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
		}}
	}

	occurrences := rule.Between(rangeStart, rangeEnd, true)
	var events []Event
	for _, occStart := range occurrences {
		events = append(events, Event{
			UID:         uid,
			Summary:     summary,
			Description: description,
			Location:    location,
			Start:       occStart,
			End:         occStart.Add(duration),
		})
	}

	return events
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

func (cal *Calendar) SupportsEvents() bool {
	if len(cal.SupportedComponentSet) == 0 {
		return true
	}
	for _, comp := range cal.SupportedComponentSet {
		if comp == "VEVENT" {
			return true
		}
	}
	return false
}

func parseICalTime(prop ical.Prop) time.Time {
	params := prop.Params
	value := prop.Value

	var loc *time.Location = time.Local
	if tzid := params.Get("TZID"); tzid != "" {
		if l, err := time.LoadLocation(tzid); err == nil {
			loc = l
		}
	}

	formats := []string{
		"20060102T150405Z",
		"20060102T150405",
		"20060102",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, value, loc); err == nil {
			if strings.HasSuffix(value, "Z") {
				return t.UTC()
			}
			return t
		}
	}

	return time.Time{}
}
