package icloud

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jingkaihe/icloud-cli/internal/caldav"
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage iCloud calendar events",
	Long:  `List, create, update, and delete iCloud calendar events.`,
}

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List calendar events",
	Long: `List calendar events for the specified date range.
	
By default, lists events for the current week. Use --start-date and --end-date to filter.

Date formats supported:
  - YYYY-MM-DD (e.g., 2024-01-15)
  - Relative: 1d (1 day from now), 1w (1 week from now), -2d (2 days ago)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, _, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		startStr, _ := cmd.Flags().GetString("start-date")
		endStr, _ := cmd.Flags().GetString("end-date")
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		output, _ := cmd.Flags().GetString("output")

		start, end, err := parseDateRange(startStr, endStr)
		if err != nil {
			return err
		}

		var events []caldav.Event
		if calendarID != "" {
			cal, err := client.GetCalendar(ctx, calendarID)
			if err != nil {
				return fmt.Errorf("failed to find calendar: %w", err)
			}
			events, err = client.ListEvents(ctx, cal.Path, start, end)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}
			for i := range events {
				events[i].CalendarPath = cal.Path
				events[i].CalendarName = cal.Name
			}
		} else {
			events, err = client.ListEventsAllCalendars(ctx, start, end)
			if err != nil {
				return fmt.Errorf("failed to list events: %w", err)
			}
		}

		sort.Slice(events, func(i, j int) bool {
			return events[i].Start.Before(events[j].Start)
		})

		if output == "json" {
			return outputJSON(events)
		}
		return outputTSV(events)
	},
}

func parseDateRange(startStr, endStr string) (time.Time, time.Time, error) {
	now := time.Now()

	var start, end time.Time

	if startStr == "" {
		start = startOfWeek(now)
	} else {
		t, err := parseDate(startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start-date: %w", err)
		}
		start = t
	}

	if endStr == "" {
		end = start.AddDate(0, 0, 7)
	} else {
		t, err := parseDate(endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end-date: %w", err)
		}
		end = t.AddDate(0, 0, 1)
	}

	return start, end, nil
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	re := regexp.MustCompile(`^(-?\d+)([dwmDWM])$`)
	matches := re.FindStringSubmatch(s)
	if matches != nil {
		num, _ := strconv.Atoi(matches[1])
		unit := strings.ToLower(matches[2])

		now := time.Now()
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		switch unit {
		case "d":
			return now.AddDate(0, 0, num), nil
		case "w":
			return now.AddDate(0, 0, num*7), nil
		case "m":
			return now.AddDate(0, num, 0), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s (use YYYY-MM-DD or relative like 1d, 1w, -2d)", s)
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
}

type eventOutput struct {
	UID         string `json:"uid"`
	Title       string `json:"title"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	Calendar    string `json:"calendar,omitempty"`
}

func outputJSON(events []caldav.Event) error {
	output := make([]eventOutput, len(events))
	for i, e := range events {
		output[i] = eventOutput{
			UID:         e.UID,
			Title:       e.Summary,
			Start:       e.Start.Format(time.RFC3339),
			End:         e.End.Format(time.RFC3339),
			Location:    e.Location,
			Description: e.Description,
			Calendar:    e.CalendarName,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputTSV(events []caldav.Event) error {
	if len(events) == 0 {
		fmt.Println("No events found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UID\tCalendar\tTitle\tStart\tEnd\tLocation")
	for _, e := range events {
		location := e.Location
		if location == "" {
			location = "-"
		}

		calendar := e.CalendarName
		if calendar == "" {
			calendar = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			e.UID,
			calendar,
			e.Summary,
			e.Start.Format("2006-01-02 15:04"),
			e.End.Format("2006-01-02 15:04"),
			location,
		)
	}
	return w.Flush()
}

var eventGetCmd = &cobra.Command{
	Use:   "get <event-uid>",
	Short: "Get details of a calendar event",
	Long:  `Get detailed information about a specific event by its UID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, _, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		eventUID := args[0]
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		output, _ := cmd.Flags().GetString("output")

		if calendarID == "" {
			return fmt.Errorf("--calendar-id is required")
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		event, err := client.GetEvent(ctx, cal.Path, eventUID)
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		event.CalendarName = cal.Name

		if output == "json" {
			return outputJSON([]caldav.Event{*event})
		}

		fmt.Printf("UID:         %s\n", event.UID)
		fmt.Printf("Title:       %s\n", event.Summary)
		fmt.Printf("Calendar:    %s\n", cal.Name)
		fmt.Printf("Start:       %s\n", event.Start.Format("2006-01-02 15:04"))
		fmt.Printf("End:         %s\n", event.End.Format("2006-01-02 15:04"))
		if event.Location != "" {
			fmt.Printf("Location:    %s\n", event.Location)
		}
		if event.Description != "" {
			fmt.Printf("Description: %s\n", event.Description)
		}
		return nil
	},
}

var eventCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new calendar event",
	Long: `Create a new event in the specified calendar.

Date/time formats supported:
  - YYYY-MM-DD HH:MM (e.g., 2024-01-15 14:30)
  - YYYY-MM-DDTHH:MM (e.g., 2024-01-15T14:30)

If end time is not specified, the event will be 1 hour long.

Recurrence rules (RRULE) follow the iCalendar RFC 5545 format:
  - FREQ=DAILY                          (every day)
  - FREQ=WEEKLY;BYDAY=MO,WE,FR          (every Mon, Wed, Fri)
  - FREQ=WEEKLY;INTERVAL=2              (every 2 weeks)
  - FREQ=MONTHLY;BYMONTHDAY=15          (15th of every month)
  - FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1  (every Jan 1st)
  - FREQ=DAILY;COUNT=10                 (daily for 10 occurrences)
  - FREQ=WEEKLY;UNTIL=20251231T235959Z  (weekly until end of 2025)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, _, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		title, _ := cmd.Flags().GetString("title")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		location, _ := cmd.Flags().GetString("location")
		description, _ := cmd.Flags().GetString("description")
		rrule, _ := cmd.Flags().GetString("rrule")

		if title == "" {
			return fmt.Errorf("--title is required")
		}
		if startStr == "" {
			return fmt.Errorf("--start is required")
		}
		if calendarID == "" {
			return fmt.Errorf("--calendar-id is required")
		}

		start, err := parseDateTime(startStr)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}

		var end time.Time
		if endStr == "" {
			end = start.Add(1 * time.Hour)
		} else {
			end, err = parseDateTime(endStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		event := caldav.Event{
			Summary:     title,
			Start:       start,
			End:         end,
			Location:    location,
			Description: description,
			RRULE:       rrule,
		}

		created, err := client.CreateEvent(ctx, cal.Path, event)
		if err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		fmt.Printf("Event created: %s (UID: %s)\n", created.Summary, created.UID)
		return nil
	},
}

var eventUpdateCmd = &cobra.Command{
	Use:   "update <event-uid>",
	Short: "Update an existing calendar event",
	Long: `Update an existing event. The event UID and calendar ID are required.

Only the fields you specify will be updated.

For recurring events, you can choose to:
  - Update entire series: use --series flag (default for non-recurring)
  - Update single occurrence: use --occurrence with the date/time of the occurrence

If neither --series nor --occurrence is specified for a recurring event,
you will be prompted to choose.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, _, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		eventUID := args[0]
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		series, _ := cmd.Flags().GetBool("series")
		occurrenceStr, _ := cmd.Flags().GetString("occurrence")

		if calendarID == "" {
			return fmt.Errorf("--calendar-id is required")
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		existing, err := client.GetEvent(ctx, cal.Path, eventUID)
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		isRecurring := existing.RRULE != ""

		if occurrenceStr != "" {
			if !isRecurring {
				return fmt.Errorf("--occurrence can only be used with recurring events")
			}

			occurrenceTime, err := parseDateTime(occurrenceStr)
			if err != nil {
				return fmt.Errorf("invalid occurrence time: %w", err)
			}

			updates := caldav.Event{}
			if title, _ := cmd.Flags().GetString("title"); title != "" {
				updates.Summary = title
			}
			if startStr, _ := cmd.Flags().GetString("start"); startStr != "" {
				start, err := parseDateTime(startStr)
				if err != nil {
					return fmt.Errorf("invalid start time: %w", err)
				}
				updates.Start = start
			}
			if endStr, _ := cmd.Flags().GetString("end"); endStr != "" {
				end, err := parseDateTime(endStr)
				if err != nil {
					return fmt.Errorf("invalid end time: %w", err)
				}
				updates.End = end
			}
			if cmd.Flags().Changed("location") {
				location, _ := cmd.Flags().GetString("location")
				updates.Location = location
			}
			if cmd.Flags().Changed("description") {
				description, _ := cmd.Flags().GetString("description")
				updates.Description = description
			}

			if err := client.UpdateEventOccurrence(ctx, cal.Path, eventUID, occurrenceTime, updates); err != nil {
				return fmt.Errorf("failed to update occurrence: %w", err)
			}

			fmt.Printf("Occurrence on %s updated.\n", occurrenceTime.Format("2006-01-02 15:04"))
			return nil
		}

		if isRecurring && !series {
			fmt.Printf("'%s' is a recurring event.\n", existing.Summary)
			fmt.Println("Use --series to update the entire series, or --occurrence <datetime> to update a single occurrence.")
			return nil
		}

		if title, _ := cmd.Flags().GetString("title"); title != "" {
			existing.Summary = title
		}
		if startStr, _ := cmd.Flags().GetString("start"); startStr != "" {
			start, err := parseDateTime(startStr)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
			existing.Start = start
		}
		if endStr, _ := cmd.Flags().GetString("end"); endStr != "" {
			end, err := parseDateTime(endStr)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
			existing.End = end
		}
		if cmd.Flags().Changed("location") {
			location, _ := cmd.Flags().GetString("location")
			existing.Location = location
		}
		if cmd.Flags().Changed("description") {
			description, _ := cmd.Flags().GetString("description")
			existing.Description = description
		}
		if cmd.Flags().Changed("rrule") {
			rrule, _ := cmd.Flags().GetString("rrule")
			existing.RRULE = rrule
		}

		if err := client.UpdateEvent(ctx, cal.Path, *existing); err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}

		if isRecurring {
			fmt.Printf("Recurring event series updated: %s\n", existing.Summary)
		} else {
			fmt.Printf("Event updated: %s\n", existing.Summary)
		}
		return nil
	},
}

var eventDeleteCmd = &cobra.Command{
	Use:   "delete <event-uid>",
	Short: "Delete a calendar event",
	Long: `Delete an event from the specified calendar.

For recurring events, you can choose to:
  - Delete entire series: use --series flag
  - Delete single occurrence: use --occurrence with the date/time of the occurrence

If neither --series nor --occurrence is specified for a recurring event,
you will be prompted to choose.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, _, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		eventUID := args[0]
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		force, _ := cmd.Flags().GetBool("force")
		series, _ := cmd.Flags().GetBool("series")
		occurrenceStr, _ := cmd.Flags().GetString("occurrence")

		if calendarID == "" {
			return fmt.Errorf("--calendar-id is required")
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		existing, err := client.GetEvent(ctx, cal.Path, eventUID)
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		isRecurring := existing.RRULE != ""

		if occurrenceStr != "" {
			if !isRecurring {
				return fmt.Errorf("--occurrence can only be used with recurring events")
			}
			occurrenceTime, err := parseDateTime(occurrenceStr)
			if err != nil {
				return fmt.Errorf("invalid occurrence time: %w", err)
			}

			if !force {
				fmt.Printf("Are you sure you want to delete the occurrence of '%s' on %s? Use --force to confirm.\n",
					existing.Summary, occurrenceTime.Format("2006-01-02 15:04"))
				return nil
			}

			if err := client.DeleteEventOccurrence(ctx, cal.Path, eventUID, occurrenceTime); err != nil {
				return fmt.Errorf("failed to delete occurrence: %w", err)
			}

			fmt.Printf("Occurrence on %s deleted.\n", occurrenceTime.Format("2006-01-02 15:04"))
			return nil
		}

		if isRecurring && !series {
			fmt.Printf("'%s' is a recurring event.\n", existing.Summary)
			fmt.Println("Use --series to delete the entire series, or --occurrence <datetime> to delete a single occurrence.")
			return nil
		}

		if !force {
			if isRecurring {
				fmt.Printf("Are you sure you want to delete the entire series '%s'? Use --force to confirm.\n", existing.Summary)
			} else {
				fmt.Printf("Are you sure you want to delete '%s'? Use --force to confirm.\n", existing.Summary)
			}
			return nil
		}

		if err := client.DeleteEvent(ctx, cal.Path, eventUID); err != nil {
			return fmt.Errorf("failed to delete event: %w", err)
		}

		if isRecurring {
			fmt.Println("Recurring event series deleted.")
		} else {
			fmt.Println("Event deleted.")
		}
		return nil
	},
}

func parseDateTime(s string) (time.Time, error) {
	s = strings.ReplaceAll(s, "T", " ")

	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid datetime format: %s (use YYYY-MM-DD HH:MM)", s)
}

func init() {
	eventCmd.PersistentFlags().StringP("account", "a", "", "Account to use (email or alias)")

	eventListCmd.Flags().StringP("start-date", "s", "", "Start date (YYYY-MM-DD or relative like 1d, 1w)")
	eventListCmd.Flags().StringP("end-date", "e", "", "End date (YYYY-MM-DD or relative like 1d, 1w)")
	eventListCmd.Flags().StringP("calendar-id", "c", "", "Filter by calendar (ID or name)")
	eventListCmd.Flags().StringP("output", "o", "tsv", "Output format (tsv or json)")

	eventGetCmd.Flags().StringP("calendar-id", "c", "", "Calendar ID or name (required)")
	eventGetCmd.Flags().StringP("output", "o", "", "Output format (json)")

	eventCreateCmd.Flags().StringP("title", "t", "", "Event title (required)")
	eventCreateCmd.Flags().StringP("start", "s", "", "Start time (required, format: YYYY-MM-DD HH:MM)")
	eventCreateCmd.Flags().StringP("end", "e", "", "End time (format: YYYY-MM-DD HH:MM)")
	eventCreateCmd.Flags().StringP("calendar-id", "c", "", "Calendar ID or name (required)")
	eventCreateCmd.Flags().StringP("location", "l", "", "Event location")
	eventCreateCmd.Flags().StringP("description", "d", "", "Event description")
	eventCreateCmd.Flags().StringP("rrule", "r", "", "Recurrence rule (e.g., FREQ=WEEKLY;BYDAY=MO,WE,FR)")

	eventUpdateCmd.Flags().StringP("calendar-id", "c", "", "Calendar ID or name (required)")
	eventUpdateCmd.Flags().StringP("title", "t", "", "New event title")
	eventUpdateCmd.Flags().StringP("start", "s", "", "New start time (format: YYYY-MM-DD HH:MM)")
	eventUpdateCmd.Flags().StringP("end", "e", "", "New end time (format: YYYY-MM-DD HH:MM)")
	eventUpdateCmd.Flags().StringP("location", "l", "", "New event location")
	eventUpdateCmd.Flags().StringP("description", "d", "", "New event description")
	eventUpdateCmd.Flags().StringP("rrule", "r", "", "New recurrence rule (e.g., FREQ=WEEKLY;BYDAY=MO,WE,FR)")
	eventUpdateCmd.Flags().BoolP("series", "S", false, "Update entire recurring series")
	eventUpdateCmd.Flags().StringP("occurrence", "o", "", "Update single occurrence (format: YYYY-MM-DD HH:MM)")

	eventDeleteCmd.Flags().StringP("calendar-id", "c", "", "Calendar ID or name (required)")
	eventDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	eventDeleteCmd.Flags().BoolP("series", "S", false, "Delete entire recurring series")
	eventDeleteCmd.Flags().StringP("occurrence", "o", "", "Delete single occurrence (format: YYYY-MM-DD HH:MM)")

	eventCmd.AddCommand(eventListCmd)
	eventCmd.AddCommand(eventGetCmd)
	eventCmd.AddCommand(eventCreateCmd)
	eventCmd.AddCommand(eventUpdateCmd)
	eventCmd.AddCommand(eventDeleteCmd)
}
