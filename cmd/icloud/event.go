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
	fmt.Fprintln(w, "Calendar\tTitle\tStart\tEnd\tLocation\tDescription")
	for _, e := range events {
		location := e.Location
		if location == "" {
			location = "-"
		}
		description := e.Description
		if description == "" {
			description = "-"
		}
		description = strings.ReplaceAll(description, "\n", " ")
		description = strings.ReplaceAll(description, "\t", " ")

		calendar := e.CalendarName
		if calendar == "" {
			calendar = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			calendar,
			e.Summary,
			e.Start.Format("2006-01-02 15:04"),
			e.End.Format("2006-01-02 15:04"),
			location,
			description,
		)
	}
	return w.Flush()
}

func init() {
	eventCmd.PersistentFlags().StringP("account", "a", "", "Account to use (email or alias)")

	eventListCmd.Flags().StringP("start-date", "s", "", "Start date (YYYY-MM-DD or relative like 1d, 1w)")
	eventListCmd.Flags().StringP("end-date", "e", "", "End date (YYYY-MM-DD or relative like 1d, 1w)")
	eventListCmd.Flags().StringP("calendar-id", "c", "", "Filter by calendar ID")
	eventListCmd.Flags().StringP("output", "o", "tsv", "Output format (tsv or json)")

	eventCmd.AddCommand(eventListCmd)
}
