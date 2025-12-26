package caldav

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"
)

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
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

func buildICS(event Event) string {
	now := time.Now().UTC().Format("20060102T150405Z")
	dtstart := event.Start.UTC().Format("20060102T150405Z")
	dtend := event.End.UTC().Format("20060102T150405Z")

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//icloud-cli//EN\r\n")
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
	if event.RecurrenceID != nil {
		sb.WriteString(fmt.Sprintf("RECURRENCE-ID:%s\r\n", event.RecurrenceID.UTC().Format("20060102T150405Z")))
	}
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", dtstart))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", dtend))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(event.Summary)))
	if event.Description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(event.Description)))
	}
	if event.Location != "" {
		sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(event.Location)))
	}
	if event.RRULE != "" {
		sb.WriteString(fmt.Sprintf("RRULE:%s\r\n", event.RRULE))
	}
	for _, exdate := range event.EXDATEs {
		sb.WriteString(fmt.Sprintf("EXDATE:%s\r\n", exdate.UTC().Format("20060102T150405Z")))
	}
	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

func buildICSWithException(master Event, exception Event) string {
	now := time.Now().UTC().Format("20060102T150405Z")

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//icloud-cli//EN\r\n")

	// Master event
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s\r\n", master.UID))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", master.Start.UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", master.End.UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(master.Summary)))
	if master.Description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(master.Description)))
	}
	if master.Location != "" {
		sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(master.Location)))
	}
	if master.RRULE != "" {
		sb.WriteString(fmt.Sprintf("RRULE:%s\r\n", master.RRULE))
	}
	for _, exdate := range master.EXDATEs {
		sb.WriteString(fmt.Sprintf("EXDATE:%s\r\n", exdate.UTC().Format("20060102T150405Z")))
	}
	sb.WriteString("END:VEVENT\r\n")

	// Exception event (modified occurrence)
	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s\r\n", exception.UID))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
	if exception.RecurrenceID != nil {
		sb.WriteString(fmt.Sprintf("RECURRENCE-ID:%s\r\n", exception.RecurrenceID.UTC().Format("20060102T150405Z")))
	}
	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", exception.Start.UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", exception.End.UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(exception.Summary)))
	if exception.Description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(exception.Description)))
	}
	if exception.Location != "" {
		sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(exception.Location)))
	}
	sb.WriteString("END:VEVENT\r\n")

	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}
