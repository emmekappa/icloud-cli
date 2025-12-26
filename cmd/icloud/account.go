package icloud

import (
	"fmt"
	"os"

	"github.com/jingkaihe/icloud-cli/internal/caldav"
	"github.com/jingkaihe/icloud-cli/internal/config"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage iCloud accounts",
	Long:  `Add, remove, list, and manage iCloud accounts for calendar access.`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Accounts) == 0 {
			fmt.Println("No accounts configured. Use 'icloud account add' to add an account.")
			return nil
		}

		fmt.Println("Configured accounts:")
		for _, a := range cfg.Accounts {
			defaultMarker := ""
			if a.IsDefault {
				defaultMarker = " (default)"
			}
			fmt.Printf("  - %s <%s> [%s]%s\n", a.Name, a.Email, a.GetAlias(), defaultMarker)
		}

		return nil
	},
}

var accountAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new iCloud account",
	Long: `Add a new iCloud account using environment variables or flags.
	
Environment variables:
  APPLE_EMAIL                   - Apple ID email address
  APPLE_APP_SPECIFIC_PASSWORD   - App-specific password`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		name, _ := cmd.Flags().GetString("name")
		alias, _ := cmd.Flags().GetString("alias")

		if email == "" {
			email = os.Getenv("APPLE_EMAIL")
		}
		if password == "" {
			password = os.Getenv("APPLE_APP_SPECIFIC_PASSWORD")
		}

		if email == "" || password == "" {
			return fmt.Errorf("email and password are required. Set APPLE_EMAIL and APPLE_APP_SPECIFIC_PASSWORD environment variables or use --email and --password flags")
		}

		if name == "" {
			name = email
		}

		fmt.Println("Verifying credentials with iCloud...")
		client, err := caldav.NewClient(email, password)
		if err != nil {
			return fmt.Errorf("failed to connect to iCloud: %w", err)
		}

		ctx := signalContext()
		_, err = client.FindCalendarHomeSet(ctx)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		account := config.Account{
			Name:     name,
			Email:    email,
			Password: password,
			Alias:    alias,
		}

		if err := cfg.AddAccount(account); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}

		fmt.Printf("Successfully added account: %s <%s> [%s]\n", name, email, account.GetAlias())
		return nil
	},
}

var accountSetDefaultCmd = &cobra.Command{
	Use:   "set-default <account>",
	Short: "Set the default account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.SetDefault(args[0]); err != nil {
			return fmt.Errorf("failed to set default account: %w", err)
		}

		fmt.Printf("Default account set to: %s\n", args[0])
		return nil
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove <account>",
	Short: "Remove an account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := cfg.RemoveAccount(args[0]); err != nil {
			return fmt.Errorf("failed to remove account: %w", err)
		}

		fmt.Printf("Account removed: %s\n", args[0])
		return nil
	},
}

func init() {
	accountAddCmd.Flags().StringP("email", "e", "", "Apple ID email address")
	accountAddCmd.Flags().StringP("password", "p", "", "App-specific password")
	accountAddCmd.Flags().StringP("name", "n", "", "Account display name")
	accountAddCmd.Flags().StringP("alias", "a", "", "Account alias (defaults to email prefix)")

	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountSetDefaultCmd)
	accountCmd.AddCommand(accountRemoveCmd)
}
