package icloud

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jingkaihe/icloud-cli/internal/config"
	"github.com/jingkaihe/icloud-cli/internal/email"
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Manage iCloud email",
	Long:  `Manage iCloud email including listing, reading, sending, and organizing messages.`,
}

var emailMailboxCmd = &cobra.Command{
	Use:   "mailbox",
	Short: "Manage mailboxes",
	Long:  `List and manage iCloud email mailboxes.`,
}

var emailMailboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all mailboxes",
	Long:  `List all available mailboxes/folders in your iCloud email account.`,
	RunE:  runMailboxList,
}

var (
	emailAccountFlag string
	emailOutputFlag  string
)

func init() {
	emailCmd.AddCommand(emailMailboxCmd)
	emailMailboxCmd.AddCommand(emailMailboxListCmd)

	emailCmd.PersistentFlags().StringVarP(&emailAccountFlag, "account", "a", "", "Account to use (default: default account)")
	emailCmd.PersistentFlags().StringVarP(&emailOutputFlag, "output", "o", "tsv", "Output format (tsv or json)")
}

func runMailboxList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	account, err := cfg.GetAccountOrDefault(emailAccountFlag)
	if err != nil {
		return err
	}

	client := email.NewClient(account.Email, account.Password)
	mailboxes, err := client.ListMailboxes()
	if err != nil {
		return err
	}

	if emailOutputFlag == "json" {
		return outputEmailAsJSON(mailboxes)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tATTRIBUTES")
	for _, mb := range mailboxes {
		fmt.Fprintf(w, "%s\t%s\n", mb.Name, mb.Attributes)
	}
	return w.Flush()
}

func outputEmailAsJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func getEmailClient(accountID string) (*email.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	account, err := cfg.GetAccountOrDefault(accountID)
	if err != nil {
		return nil, err
	}

	return email.NewClient(account.Email, account.Password), nil
}
