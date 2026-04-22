package icloud

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jingkaihe/icloud-cli/internal/reminders"
	"github.com/spf13/cobra"
)

var reminderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new reminder",
	Example: `  icloud reminder create -t "Buy milk"
  icloud reminder create -t "Call Alice" -L "ABCD-..." -d "2026-05-01 18:00" -p high
  icloud reminder create -t "Tax return" -d 2026-11-10 -n "Form 740 + receipts"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		if strings.TrimSpace(title) == "" {
			return fmt.Errorf("title is required (use -t)")
		}

		in := reminders.CreateInput{Title: title}

		if cmd.Flags().Changed("list") {
			v, _ := cmd.Flags().GetString("list")
			in.ListID = v
		}
		if cmd.Flags().Changed("notes") {
			v, _ := cmd.Flags().GetString("notes")
			in.Notes = v
		}
		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			in.URL = v
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetString("priority")
			p, err := parsePriority(v)
			if err != nil {
				return err
			}
			in.Priority = &p
		}
		if cmd.Flags().Changed("due") {
			v, _ := cmd.Flags().GetString("due")
			due, err := parseDue(v)
			if err != nil {
				return err
			}
			in.Due = due
		}

		r, err := reminders.New().Create(in)
		if err != nil {
			return err
		}
		return printReminderResult(cmd, r)
	},
}

var reminderUpdateCmd = &cobra.Command{
	Use:   "update <uid>",
	Short: "Update an existing reminder",
	Long: `Update a reminder by UID. Only the flags you provide are changed.

To clear the due date use --clear-due (ignored if --due is also given).
To clear notes or url, pass them as an empty string (-n "" or -u "").`,
	Example: `  icloud reminder update 2AC43... -t "Buy 2% milk"
  icloud reminder update 2AC43... -d "2026-05-01 20:00"
  icloud reminder update 2AC43... --clear-due
  icloud reminder update 2AC43... -L "ABCD-..."
  icloud reminder update 2AC43... -p none`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uid := args[0]
		var in reminders.UpdateInput

		if cmd.Flags().Changed("title") {
			v, _ := cmd.Flags().GetString("title")
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			in.Title = &v
		}
		if cmd.Flags().Changed("list") {
			v, _ := cmd.Flags().GetString("list")
			in.ListID = &v
		}
		if cmd.Flags().Changed("notes") {
			v, _ := cmd.Flags().GetString("notes")
			in.Notes = &v
		}
		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			in.URL = &v
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetString("priority")
			p, err := parsePriority(v)
			if err != nil {
				return err
			}
			in.Priority = &p
		}
		if cmd.Flags().Changed("due") {
			v, _ := cmd.Flags().GetString("due")
			due, err := parseDue(v)
			if err != nil {
				return err
			}
			in.Due = due
		}
		if cmd.Flags().Changed("clear-due") {
			in.ClearDue, _ = cmd.Flags().GetBool("clear-due")
		}

		r, err := reminders.New().Update(uid, in)
		if err != nil {
			return err
		}
		return printReminderResult(cmd, r)
	},
}

var reminderCompleteCmd = &cobra.Command{
	Use:     "complete <uid>",
	Short:   "Mark a reminder as completed",
	Args:    cobra.ExactArgs(1),
	Example: "  icloud reminder complete 2AC4365A-...",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := reminders.New().SetCompleted(args[0], true)
		if err != nil {
			return err
		}
		return printReminderResult(cmd, r)
	},
}

var reminderUncompleteCmd = &cobra.Command{
	Use:     "uncomplete <uid>",
	Short:   "Mark a completed reminder as not completed",
	Args:    cobra.ExactArgs(1),
	Example: "  icloud reminder uncomplete 2AC4365A-...",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := reminders.New().SetCompleted(args[0], false)
		if err != nil {
			return err
		}
		return printReminderResult(cmd, r)
	},
}

func parsePriority(s string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none":
		return 0, nil
	case "high":
		return 1, nil
	case "medium", "med":
		return 5, nil
	case "low":
		return 9, nil
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 9 {
		return n, nil
	}
	return 0, fmt.Errorf("invalid priority %q (use none|high|medium|low or 0-9)", s)
}

// parseDue accepts "YYYY-MM-DD", "YYYY-MM-DD HH:MM", "YYYY-MM-DDTHH:MM",
// or with seconds. Returns nil on empty input.
func parseDue(s string) (*reminders.DueInput, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	normalized := strings.ReplaceAll(s, "T", " ")

	dateTimeFormats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}
	for _, f := range dateTimeFormats {
		if t, err := time.ParseInLocation(f, normalized, time.Local); err == nil {
			hour := t.Hour()
			minute := t.Minute()
			return &reminders.DueInput{
				Year: t.Year(), Month: int(t.Month()), Day: t.Day(),
				Hour: &hour, Minute: &minute,
			}, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", normalized, time.Local); err == nil {
		return &reminders.DueInput{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}, nil
	}
	return nil, fmt.Errorf("invalid due format %q (use YYYY-MM-DD or YYYY-MM-DD HH:MM)", s)
}

func printReminderResult(cmd *cobra.Command, r *reminders.Reminder) error {
	format, _ := cmd.Flags().GetString("output")
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	return renderReminders([]reminders.Reminder{*r}, format)
}

func init() {
	reminderCreateCmd.Flags().StringP("title", "t", "", "Reminder title (required)")
	reminderCreateCmd.Flags().StringP("list", "L", "", "Target reminder list ID (default: system default list)")
	reminderCreateCmd.Flags().StringP("due", "d", "", "Due date: YYYY-MM-DD or YYYY-MM-DD HH:MM")
	reminderCreateCmd.Flags().StringP("priority", "p", "", "Priority: none | high | medium | low (or 0-9)")
	reminderCreateCmd.Flags().StringP("notes", "n", "", "Notes / description")
	reminderCreateCmd.Flags().StringP("url", "u", "", "Associated URL")

	reminderUpdateCmd.Flags().StringP("title", "t", "", "New title")
	reminderUpdateCmd.Flags().StringP("list", "L", "", "Move reminder to this list ID")
	reminderUpdateCmd.Flags().StringP("due", "d", "", "New due date")
	reminderUpdateCmd.Flags().StringP("priority", "p", "", "Priority: none | high | medium | low (or 0-9)")
	reminderUpdateCmd.Flags().StringP("notes", "n", "", "New notes (empty string clears)")
	reminderUpdateCmd.Flags().StringP("url", "u", "", "New URL (empty string clears)")
	reminderUpdateCmd.Flags().Bool("clear-due", false, "Remove the due date")

	reminderCmd.AddCommand(reminderCreateCmd)
	reminderCmd.AddCommand(reminderUpdateCmd)
	reminderCmd.AddCommand(reminderCompleteCmd)
	reminderCmd.AddCommand(reminderUncompleteCmd)
}
