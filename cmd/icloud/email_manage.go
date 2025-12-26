package icloud

import (
	"fmt"

	"github.com/spf13/cobra"
)

var emailMoveCmd = &cobra.Command{
	Use:   "move <uid>",
	Short: "Move an email to another mailbox",
	Long: `Move an email from one mailbox to another.

Examples:
  icloud email move 12345 -m INBOX -d Archive
  icloud email move 12345 -m INBOX -d "Sent Messages"`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailMove,
}

var emailDeleteCmd = &cobra.Command{
	Use:   "delete <uid>",
	Short: "Delete an email",
	Long: `Delete an email by moving it to Trash.

Examples:
  icloud email delete 12345 -m INBOX
  icloud email delete 12345 -m INBOX -f`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailDelete,
}

var emailMarkCmd = &cobra.Command{
	Use:   "mark <uid>",
	Short: "Mark email as read or unread",
	Long: `Mark an email as read or unread.

Examples:
  icloud email mark 12345 -m INBOX --read
  icloud email mark 12345 -m INBOX --unread`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailMark,
}

var emailFlagCmd = &cobra.Command{
	Use:   "flag <uid>",
	Short: "Flag or unflag an email",
	Long: `Flag (star) or unflag an email.

Examples:
  icloud email flag 12345 -m INBOX --set
  icloud email flag 12345 -m INBOX --unset`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailFlag,
}

var (
	moveSourceFlag string
	moveDestFlag   string
	deleteForceFlag bool
	markReadFlag   bool
	markUnreadFlag bool
	flagSetFlag    bool
	flagUnsetFlag  bool
)

func init() {
	emailCmd.AddCommand(emailMoveCmd)
	emailCmd.AddCommand(emailDeleteCmd)
	emailCmd.AddCommand(emailMarkCmd)
	emailCmd.AddCommand(emailFlagCmd)

	emailMoveCmd.Flags().StringVarP(&moveSourceFlag, "mailbox", "m", "INBOX", "Source mailbox")
	emailMoveCmd.Flags().StringVarP(&moveDestFlag, "destination", "d", "", "Destination mailbox")
	emailMoveCmd.MarkFlagRequired("destination")

	emailDeleteCmd.Flags().StringVarP(&moveSourceFlag, "mailbox", "m", "INBOX", "Mailbox containing the email")
	emailDeleteCmd.Flags().BoolVarP(&deleteForceFlag, "force", "f", false, "Skip confirmation")

	emailMarkCmd.Flags().StringVarP(&moveSourceFlag, "mailbox", "m", "INBOX", "Mailbox containing the email")
	emailMarkCmd.Flags().BoolVar(&markReadFlag, "read", false, "Mark as read")
	emailMarkCmd.Flags().BoolVar(&markUnreadFlag, "unread", false, "Mark as unread")

	emailFlagCmd.Flags().StringVarP(&moveSourceFlag, "mailbox", "m", "INBOX", "Mailbox containing the email")
	emailFlagCmd.Flags().BoolVar(&flagSetFlag, "set", false, "Flag the email")
	emailFlagCmd.Flags().BoolVar(&flagUnsetFlag, "unset", false, "Unflag the email")
}

func runEmailMove(cmd *cobra.Command, args []string) error {
	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	if err := client.MoveEmail(moveSourceFlag, uid, moveDestFlag); err != nil {
		return fmt.Errorf("failed to move email: %w", err)
	}

	fmt.Printf("Email moved to %s\n", moveDestFlag)
	return nil
}

func runEmailDelete(cmd *cobra.Command, args []string) error {
	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	if !deleteForceFlag {
		fmt.Printf("Are you sure you want to delete email %d? [y/N] ", uid)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	if err := client.DeleteEmail(moveSourceFlag, uid); err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}

	fmt.Println("Email moved to Trash")
	return nil
}

func runEmailMark(cmd *cobra.Command, args []string) error {
	if !markReadFlag && !markUnreadFlag {
		return fmt.Errorf("either --read or --unread must be specified")
	}
	if markReadFlag && markUnreadFlag {
		return fmt.Errorf("cannot specify both --read and --unread")
	}

	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	if err := client.MarkEmail(moveSourceFlag, uid, markReadFlag); err != nil {
		return fmt.Errorf("failed to mark email: %w", err)
	}

	if markReadFlag {
		fmt.Println("Email marked as read")
	} else {
		fmt.Println("Email marked as unread")
	}
	return nil
}

func runEmailFlag(cmd *cobra.Command, args []string) error {
	if !flagSetFlag && !flagUnsetFlag {
		return fmt.Errorf("either --set or --unset must be specified")
	}
	if flagSetFlag && flagUnsetFlag {
		return fmt.Errorf("cannot specify both --set and --unset")
	}

	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	if err := client.FlagEmail(moveSourceFlag, uid, flagSetFlag); err != nil {
		return fmt.Errorf("failed to flag email: %w", err)
	}

	if flagSetFlag {
		fmt.Println("Email flagged")
	} else {
		fmt.Println("Email unflagged")
	}
	return nil
}
