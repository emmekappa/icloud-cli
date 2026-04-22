package icloud

import (
	"github.com/spf13/cobra"
)

var reminderCmd = &cobra.Command{
	Use:   "reminder",
	Short: "Manage iCloud Reminders (macOS only)",
	Long: `Manage iCloud Reminders.

Reminders are accessed through Apple's EventKit framework, which is only
available on macOS. The command uses the system iCloud session — no
credentials are required and app-specific passwords are NOT used, because
Apple does not expose modern Reminders via CalDAV. Data flows through the
local 'remindd' daemon that syncs with iCloud via Apple's private CloudKit
container.

On first run, macOS will prompt for authorization. You can also open
System Settings > Privacy & Security > Reminders to manage access.`,
}

func init() {
	reminderCmd.PersistentFlags().StringP("output", "o", "tsv", "Output format: tsv or json (get: text or json)")
}
