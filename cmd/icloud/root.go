package icloud

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "icloud",
	Short: "CLI tool to interact with iCloud services",
	Long:  `A command line interface for managing iCloud calendars and events.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(calendarCmd)
	rootCmd.AddCommand(eventCmd)
	rootCmd.AddCommand(emailCmd)
}
