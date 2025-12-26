package icloud

import (
	"fmt"
	"strings"

	"github.com/jingkaihe/icloud-cli/internal/config"
	"github.com/jingkaihe/icloud-cli/internal/email"
	"github.com/spf13/cobra"
)

var emailSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an email",
	Long: `Send a new email.

Examples:
  icloud email send -t "john@example.com" -s "Hello" -B "Message body"
  icloud email send -t "john@example.com,jane@example.com" -s "Report" -f ./report.html
  icloud email send -t "john@example.com" -s "FYI" -c "manager@example.com" -B "See attached"`,
	RunE: runEmailSend,
}

var emailReplyCmd = &cobra.Command{
	Use:   "reply <uid>",
	Short: "Reply to an email",
	Long: `Reply to an email by its UID.

Examples:
  icloud email reply 12345 -m INBOX -B "Thanks for your email!"
  icloud email reply 12345 -m INBOX --all -B "Thanks everyone!"`,
	Args: cobra.ExactArgs(1),
	RunE: runEmailReply,
}

var (
	sendToFlag      string
	sendCCFlag      string
	sendBCCFlag     string
	sendSubjectFlag string
	sendBodyFlag    string
	sendFileFlag    string
	replyAllFlag    bool
	replyMailboxFlag string
)

func init() {
	emailCmd.AddCommand(emailSendCmd)
	emailCmd.AddCommand(emailReplyCmd)

	emailSendCmd.Flags().StringVarP(&sendToFlag, "to", "t", "", "Recipient addresses (comma-separated)")
	emailSendCmd.Flags().StringVarP(&sendCCFlag, "cc", "c", "", "CC addresses (comma-separated)")
	emailSendCmd.Flags().StringVarP(&sendBCCFlag, "bcc", "b", "", "BCC addresses (comma-separated)")
	emailSendCmd.Flags().StringVarP(&sendSubjectFlag, "subject", "s", "", "Email subject")
	emailSendCmd.Flags().StringVarP(&sendBodyFlag, "body", "B", "", "Email body text")
	emailSendCmd.Flags().StringVarP(&sendFileFlag, "file", "f", "", "Read body from file (.txt or .html)")
	emailSendCmd.MarkFlagRequired("to")
	emailSendCmd.MarkFlagRequired("subject")

	emailReplyCmd.Flags().StringVarP(&replyMailboxFlag, "mailbox", "m", "INBOX", "Mailbox containing the original email")
	emailReplyCmd.Flags().StringVarP(&sendBodyFlag, "body", "B", "", "Reply body text")
	emailReplyCmd.Flags().StringVarP(&sendFileFlag, "file", "f", "", "Read body from file")
	emailReplyCmd.Flags().BoolVar(&replyAllFlag, "all", false, "Reply to all recipients")
}

func runEmailSend(cmd *cobra.Command, args []string) error {
	if sendBodyFlag == "" && sendFileFlag == "" {
		return fmt.Errorf("either --body or --file must be specified")
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	opts := email.SendOptions{
		To:       parseAddresses(sendToFlag),
		CC:       parseAddresses(sendCCFlag),
		BCC:      parseAddresses(sendBCCFlag),
		Subject:  sendSubjectFlag,
		Body:     sendBodyFlag,
		BodyFile: sendFileFlag,
	}

	if err := client.SendEmail(opts); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully!")
	return nil
}

func runEmailReply(cmd *cobra.Command, args []string) error {
	if sendBodyFlag == "" && sendFileFlag == "" {
		return fmt.Errorf("either --body or --file must be specified")
	}

	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	account, err := cfg.GetAccountOrDefault(emailAccountFlag)
	if err != nil {
		return err
	}

	client := email.NewClient(account.Email, account.Password)

	original, err := client.GetEmail(replyMailboxFlag, uid)
	if err != nil {
		return fmt.Errorf("failed to get original email: %w", err)
	}

	reply := email.BuildReplyEmail(original, replyAllFlag, account.Email)

	to := make([]string, len(reply.To))
	for i, addr := range reply.To {
		to[i] = addr.Address
	}

	cc := make([]string, len(reply.CC))
	for i, addr := range reply.CC {
		cc[i] = addr.Address
	}

	opts := email.SendOptions{
		To:         to,
		CC:         cc,
		Subject:    reply.Subject,
		Body:       sendBodyFlag,
		BodyFile:   sendFileFlag,
		InReplyTo:  reply.InReplyTo,
		References: reply.References,
	}

	if err := client.SendEmail(opts); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	fmt.Println("Reply sent successfully!")
	return nil
}

func parseAddresses(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseUID(s string) (uint32, error) {
	var uid uint64
	_, err := fmt.Sscanf(s, "%d", &uid)
	if err != nil {
		return 0, fmt.Errorf("invalid UID: %s", s)
	}
	return uint32(uid), nil
}
