package icloud

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/jingkaihe/icloud-cli/internal/caldav"
	"github.com/jingkaihe/icloud-cli/internal/config"
	"github.com/spf13/cobra"
)

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Manage iCloud calendars",
	Long:  `List, create, update, and delete iCloud calendars.`,
}

func getCalDAVClient(cmd *cobra.Command) (*caldav.Client, *config.Account, error) {
	accountFlag, _ := cmd.Flags().GetString("account")

	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	account, err := cfg.GetAccountOrDefault(accountFlag)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get account: %w\nUse 'icloud account add' to add an account", err)
	}

	client, err := caldav.NewClient(account.Email, account.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to iCloud: %w", err)
	}

	return client, account, nil
}

func signalContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	return ctx
}

var calendarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all calendars",
	Long:  `List all calendars for the specified account (or default account if not specified).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()

		client, account, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		calendars, err := client.ListCalendars(ctx)
		if err != nil {
			return fmt.Errorf("failed to list calendars: %w", err)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found.")
			return nil
		}

		fmt.Printf("Calendars for %s [%s]:\n", account.Email, account.GetAlias())
		for _, cal := range calendars {
			desc := ""
			if cal.Description != "" {
				desc = fmt.Sprintf(" - %s", cal.Description)
			}
			fmt.Printf("  - %s%s\n", cal.Name, desc)
			fmt.Printf("    ID: %s\n", cal.Path)
		}

		return nil
	},
}

var calendarCreateCmd = &cobra.Command{
	Use:   "create <calendar-name>",
	Short: "Create a new calendar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()
		calendarName := args[0]

		client, account, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Creating calendar '%s' for %s...\n", calendarName, account.Email)

		calendar, err := client.CreateCalendar(ctx, calendarName)
		if err != nil {
			return fmt.Errorf("failed to create calendar: %w", err)
		}

		fmt.Printf("Successfully created calendar: %s\n", calendar.Name)
		fmt.Printf("  ID: %s\n", calendar.Path)

		return nil
	},
}

var calendarDeleteCmd = &cobra.Command{
	Use:   "delete <calendar-id>",
	Short: "Delete a calendar",
	Long: `Delete a calendar by its ID (path).
	
Use 'icloud calendar list' to find the calendar ID.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()
		calendarID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		client, account, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		if !force {
			fmt.Printf("Are you sure you want to delete calendar '%s' from %s?\n", cal.Name, account.Email)
			fmt.Printf("This action cannot be undone. Use --force to skip this prompt.\n")
			return fmt.Errorf("operation cancelled (use --force to confirm)")
		}

		fmt.Printf("Deleting calendar '%s'...\n", cal.Name)

		if err := client.DeleteCalendar(ctx, cal.Path); err != nil {
			return fmt.Errorf("failed to delete calendar: %w", err)
		}

		fmt.Printf("Successfully deleted calendar: %s\n", cal.Name)

		return nil
	},
}

var calendarUpdateCmd = &cobra.Command{
	Use:   "update <calendar-id>",
	Short: "Update a calendar's name",
	Long: `Update a calendar's display name.
	
Use 'icloud calendar list' to find the calendar ID.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signalContext()
		calendarID := args[0]
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			return fmt.Errorf("--name flag is required")
		}

		client, account, err := getCalDAVClient(cmd)
		if err != nil {
			return err
		}

		cal, err := client.GetCalendar(ctx, calendarID)
		if err != nil {
			return fmt.Errorf("failed to find calendar: %w", err)
		}

		fmt.Printf("Updating calendar '%s' to '%s' for %s...\n", cal.Name, name, account.Email)

		if err := client.UpdateCalendar(ctx, cal.Path, name); err != nil {
			return fmt.Errorf("failed to update calendar: %w", err)
		}

		fmt.Printf("Successfully updated calendar: %s -> %s\n", cal.Name, name)

		return nil
	},
}

func init() {
	calendarCmd.PersistentFlags().StringP("account", "a", "", "Account to use (email or alias)")

	calendarDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	calendarUpdateCmd.Flags().StringP("name", "n", "", "New calendar name")

	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarCreateCmd)
	calendarCmd.AddCommand(calendarDeleteCmd)
	calendarCmd.AddCommand(calendarUpdateCmd)
}
