package icloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/jingkaihe/icloud-cli/internal/reminders"
	"github.com/spf13/cobra"
)

var reminderSearchCmd = &cobra.Command{
	Use:   "search [query...]",
	Short: "Search reminders by text, list, priority, or due date",
	Long: `Search reminders across all lists (or a specific one).

[query...] is matched case-insensitively against each reminder's title and
notes. Multiple terms are AND-ed: every term must appear somewhere in the
title or the notes. If no query terms are given, all reminders that match
the flag filters are returned.

Flag filters compose with the text query:
  -L, --list              restrict to a single list (ID)
      --include-completed include completed reminders (default: incomplete only)
  -p, --priority          filter by priority label: none | high | medium | low
      --due-before DATE   only reminders with a due date strictly before
      --due-after DATE    only reminders with a due date strictly after
      --no-due            only reminders without a due date
  -n, --limit N           cap the number of results`,
	Example: `  icloud reminder search milk
  icloud reminder search "tax" -p high
  icloud reminder search --due-before 2026-12-31
  icloud reminder search anagrafe -L 8C8FC09C-A89D-4035-B882-D5DEE85DCEAF
  icloud reminder search --no-due -L 2255CF62-DBA1-4444-AB49-479ABB9FACC4`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		listID, _ := cmd.Flags().GetString("list")
		includeCompleted, _ := cmd.Flags().GetBool("include-completed")
		format, _ := cmd.Flags().GetString("output")
		limit, _ := cmd.Flags().GetInt("limit")
		priority, _ := cmd.Flags().GetString("priority")
		dueBefore, _ := cmd.Flags().GetString("due-before")
		dueAfter, _ := cmd.Flags().GetString("due-after")
		noDue, _ := cmd.Flags().GetBool("no-due")

		var priorityLabel string
		if cmd.Flags().Changed("priority") {
			p := strings.ToLower(strings.TrimSpace(priority))
			switch p {
			case "none", "high", "medium", "low":
				priorityLabel = p
			default:
				return fmt.Errorf("invalid --priority %q (use none|high|medium|low)", priority)
			}
		}

		var before, after *time.Time
		if dueBefore != "" {
			t, err := time.ParseInLocation("2006-01-02", dueBefore, time.Local)
			if err != nil {
				return fmt.Errorf("invalid --due-before %q (use YYYY-MM-DD)", dueBefore)
			}
			before = &t
		}
		if dueAfter != "" {
			t, err := time.ParseInLocation("2006-01-02", dueAfter, time.Local)
			if err != nil {
				return fmt.Errorf("invalid --due-after %q (use YYYY-MM-DD)", dueAfter)
			}
			after = &t
		}

		terms := make([]string, 0, len(args))
		for _, a := range args {
			if s := strings.ToLower(strings.TrimSpace(a)); s != "" {
				terms = append(terms, s)
			}
		}

		items, err := reminders.New().ListReminders(listID, includeCompleted)
		if err != nil {
			return err
		}

		filtered := make([]reminders.Reminder, 0, len(items))
		for _, r := range items {
			if !matchesTerms(r, terms) {
				continue
			}
			if priorityLabel != "" && reminders.PriorityLabel(r.Priority) != priorityLabel {
				continue
			}
			if noDue && r.Due != "" {
				continue
			}
			if before != nil || after != nil {
				rt, ok := parseReminderDue(r.Due)
				if !ok {
					continue
				}
				if before != nil && !rt.Before(*before) {
					continue
				}
				if after != nil && !rt.After(*after) {
					continue
				}
			}
			filtered = append(filtered, r)
		}

		if limit > 0 && len(filtered) > limit {
			filtered = filtered[:limit]
		}
		return renderReminders(filtered, format)
	},
}

func matchesTerms(r reminders.Reminder, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	hay := strings.ToLower(r.Title + "\n" + r.Notes)
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

func init() {
	reminderSearchCmd.Flags().StringP("list", "L", "", "Restrict to a single reminder list ID")
	reminderSearchCmd.Flags().Bool("include-completed", false, "Include completed reminders")
	reminderSearchCmd.Flags().StringP("priority", "p", "", "Filter by priority: none | high | medium | low")
	reminderSearchCmd.Flags().String("due-before", "", "Only reminders due before YYYY-MM-DD")
	reminderSearchCmd.Flags().String("due-after", "", "Only reminders due after YYYY-MM-DD")
	reminderSearchCmd.Flags().Bool("no-due", false, "Only reminders without a due date")
	reminderSearchCmd.Flags().IntP("limit", "n", 0, "Max number of results (0 = no limit)")
	reminderSearchCmd.Flags().StringP("output", "o", "tsv", "Output format: tsv or json")

	reminderCmd.AddCommand(reminderSearchCmd)
}
