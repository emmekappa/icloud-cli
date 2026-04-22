package icloud

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jingkaihe/icloud-cli/internal/reminders"
	"github.com/spf13/cobra"
)

var reminderListsCmd = &cobra.Command{
	Use:   "lists",
	Short: "List all reminder lists (containers)",
	Example: `  icloud reminder lists
  icloud reminder lists -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		lists, err := reminders.New().ListLists()
		if err != nil {
			return err
		}

		if format == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(lists)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tSOURCE\tCOLOR\tDEFAULT\tWRITABLE")
		for _, l := range lists {
			def := ""
			if l.IsDefault {
				def = "*"
			}
			writable := "yes"
			if !l.AllowsModifications {
				writable = "no"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", l.ID, l.Title, l.Source, l.Color, def, writable)
		}
		return w.Flush()
	},
}

var reminderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List reminders (items)",
	Long: `List reminders across all lists or filtered by a specific list.

By default only incomplete reminders are shown. Use --include-completed to
show completed ones too.`,
	Example: `  icloud reminder list
  icloud reminder list -L "ABCD-1234-..."
  icloud reminder list --include-completed
  icloud reminder list -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, _ := cmd.Flags().GetString("list")
		includeCompleted, _ := cmd.Flags().GetBool("include-completed")
		format, _ := cmd.Flags().GetString("output")
		limit, _ := cmd.Flags().GetInt("limit")

		items, err := reminders.New().ListReminders(listID, includeCompleted)
		if err != nil {
			return err
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		return renderReminders(items, format)
	},
}

func renderReminders(items []reminders.Reminder, format string) error {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UID\tLIST\tDONE\tPRIORITY\tDUE\tTITLE")
	for _, r := range items {
		done := " "
		if r.Completed {
			done = "x"
		}
		due := r.Due
		if due == "" {
			due = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.UID,
			r.List,
			done,
			reminders.PriorityLabel(r.Priority),
			due,
			singleLine(r.Title),
		)
	}
	return w.Flush()
}

// parseReminderDue interprets the string form produced by the EventKit bridge:
// "YYYY-MM-DD" (all-day) or RFC3339 with offset (timed). Returns (t, true) on
// success, (zero, false) when s is empty or unparseable.
func parseReminderDue(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, f := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var reminderGetCmd = &cobra.Command{
	Use:   "get <uid>",
	Short: "Get details of a single reminder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		r, err := reminders.New().GetReminder(args[0])
		if err != nil {
			return err
		}

		if format == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(r)
		}

		fmt.Printf("UID:       %s\n", r.UID)
		fmt.Printf("Title:     %s\n", r.Title)
		fmt.Printf("List:      %s (%s)\n", r.List, r.ListID)
		fmt.Printf("Completed: %t\n", r.Completed)
		fmt.Printf("Priority:  %s (%d)\n", reminders.PriorityLabel(r.Priority), r.Priority)
		if r.Due != "" {
			fmt.Printf("Due:       %s\n", r.Due)
		}
		if r.URL != "" {
			fmt.Printf("URL:       %s\n", r.URL)
		}
		if r.Notes != "" {
			fmt.Printf("\nNotes:\n%s\n", r.Notes)
		}
		if len(r.Alarms) > 0 {
			fmt.Printf("\nAlarms:\n")
			for _, a := range r.Alarms {
				if a.Type == "absolute" {
					fmt.Printf("  - at %s\n", a.At)
				} else {
					fmt.Printf("  - %s before due\n", formatOffset(a.OffsetSeconds))
				}
			}
		}
		if r.CreatedAt != "" {
			fmt.Printf("\nCreated:   %s\n", r.CreatedAt)
		}
		if r.ModifiedAt != "" {
			fmt.Printf("Modified:  %s\n", r.ModifiedAt)
		}
		return nil
	},
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func formatOffset(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	if d < 0 {
		d = -d
	}
	return d.String()
}

func init() {
	reminderListCmd.Flags().StringP("list", "L", "", "Reminder list ID to filter by")
	reminderListCmd.Flags().Bool("include-completed", false, "Include completed reminders")
	reminderListCmd.Flags().StringP("output", "o", "tsv", "Output format: tsv or json")
	reminderListCmd.Flags().IntP("limit", "n", 0, "Max number of reminders (0 = no limit)")

	reminderGetCmd.Flags().StringP("output", "o", "text", "Output format: text or json")

	reminderListsCmd.Flags().StringP("output", "o", "tsv", "Output format: tsv or json")

	reminderCmd.AddCommand(reminderListCmd)
	reminderCmd.AddCommand(reminderGetCmd)
	reminderCmd.AddCommand(reminderListsCmd)
}
