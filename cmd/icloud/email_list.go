package icloud

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/jingkaihe/icloud-cli/internal/email"
	"github.com/spf13/cobra"
)

var emailListCmd = &cobra.Command{
	Use:   "list",
	Short: "List emails in a mailbox",
	Long: `List emails in a mailbox (default: INBOX).

Examples:
  icloud email list
  icloud email list -m "Sent Messages"
  icloud email list -n 50 -o json`,
	RunE: runEmailList,
}

var emailGetCmd = &cobra.Command{
	Use:   "get <uid>",
	Short: "Get a specific email by UID",
	Long: `Read a specific email by its UID.

Examples:
  icloud email get 12345 -m INBOX
  icloud email get 12345 -m INBOX -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailGet,
}

var emailSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search emails",
	Long: `Search emails using IMAP search criteria.

Supported criteria:
  FROM:<address>     - Search by sender
  TO:<address>       - Search by recipient
  SUBJECT:<text>     - Search by subject
  BODY:<text>        - Search body text
  SINCE:YYYY-MM-DD   - Emails since date
  BEFORE:YYYY-MM-DD  - Emails before date
  UNSEEN             - Unread emails only
  FLAGGED            - Starred emails only

Examples:
  icloud email search "FROM:john@example.com UNSEEN"
  icloud email search "SUBJECT:invoice SINCE:2024-01-01"
  icloud email search "BODY:meeting"`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailSearch,
}

var (
	emailMailboxFlag string
	emailLimitFlag   uint32
)

func init() {
	emailCmd.AddCommand(emailListCmd)
	emailCmd.AddCommand(emailGetCmd)
	emailCmd.AddCommand(emailSearchCmd)

	emailListCmd.Flags().StringVarP(&emailMailboxFlag, "mailbox", "m", "INBOX", "Mailbox to list emails from")
	emailListCmd.Flags().Uint32VarP(&emailLimitFlag, "limit", "n", 20, "Number of emails to fetch")

	emailGetCmd.Flags().StringVarP(&emailMailboxFlag, "mailbox", "m", "INBOX", "Mailbox containing the email")

	emailSearchCmd.Flags().StringVarP(&emailMailboxFlag, "mailbox", "m", "INBOX", "Mailbox to search in")
	emailSearchCmd.Flags().Uint32VarP(&emailLimitFlag, "limit", "n", 20, "Maximum number of results")
}

func runEmailList(cmd *cobra.Command, args []string) error {
	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	emails, err := client.ListEmails(emailMailboxFlag, emailLimitFlag)
	if err != nil {
		return err
	}

	if emailOutputFlag == "json" {
		return outputEmailAsJSON(emails)
	}

	return outputEmailList(emails)
}

func runEmailGet(cmd *cobra.Command, args []string) error {
	uid, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return fmt.Errorf("invalid UID: %s", args[0])
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	emailMsg, err := client.GetEmail(emailMailboxFlag, uint32(uid))
	if err != nil {
		return err
	}

	if emailOutputFlag == "json" {
		return outputEmailAsJSON(emailMsg)
	}

	return outputEmailDetail(emailMsg)
}

func runEmailSearch(cmd *cobra.Command, args []string) error {
	criteria, err := email.ParseSearchQuery(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	emails, err := client.SearchEmails(emailMailboxFlag, criteria, emailLimitFlag)
	if err != nil {
		return err
	}

	if emailOutputFlag == "json" {
		return outputEmailAsJSON(emails)
	}

	return outputEmailList(emails)
}

func outputEmailList(emails []email.Email) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UID\tDATE\tFROM\tSUBJECT\tFLAGS")
	for _, e := range emails {
		from := ""
		if len(e.From) > 0 {
			from = e.From[0].Address
		}
		flags := ""
		if !e.Seen {
			flags += "U"
		}
		if e.Flagged {
			flags += "*"
		}
		if e.Answered {
			flags += "R"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			e.UID,
			e.Date.Format("2006-01-02 15:04"),
			truncate(from, 30),
			truncate(e.Subject, 50),
			flags,
		)
	}
	return w.Flush()
}

func outputEmailDetail(e *email.Email) error {
	fmt.Printf("Message-ID: %s\n", e.MessageID)
	fmt.Printf("UID: %d\n", e.UID)
	fmt.Printf("Date: %s\n", e.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	fmt.Printf("From: %s\n", formatAddresses(e.From))
	fmt.Printf("To: %s\n", formatAddresses(e.To))
	if len(e.CC) > 0 {
		fmt.Printf("CC: %s\n", formatAddresses(e.CC))
	}
	fmt.Printf("Subject: %s\n", e.Subject)
	fmt.Printf("Flags: seen=%t flagged=%t answered=%t\n", e.Seen, e.Flagged, e.Answered)

	if len(e.Attachments) > 0 {
		fmt.Println()
		fmt.Printf("--- Attachments (%d) ---\n", len(e.Attachments))
		for i, att := range e.Attachments {
			fmt.Printf("%d. %s (%s, %d bytes)\n", i+1, att.Filename, att.ContentType, att.Size)
			fmt.Printf("   Saved to: %s\n", att.Path)
		}
	}

	fmt.Println()
	fmt.Println("--- Body ---")
	fmt.Println(e.Body)
	return nil
}

func formatAddresses(addrs []email.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	result := ""
	for i, a := range addrs {
		if i > 0 {
			result += ", "
		}
		result += a.String()
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
