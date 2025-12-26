package caldav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
)

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

	for _, cal := range calendars {
		if strings.EqualFold(cal.Name, calendarID) {
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
