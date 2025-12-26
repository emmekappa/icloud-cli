package icloud

import (
	"fmt"

	"github.com/jingkaihe/icloud-cli/internal/email"
	"github.com/spf13/cobra"
)

var emailDraftCmd = &cobra.Command{
	Use:   "draft",
	Short: "Manage email drafts",
	Long:  `Create, list, update, delete, and send email drafts.`,
}

var emailDraftCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new draft",
	Long: `Create a new email draft.

Examples:
  icloud email draft create -t "john@example.com" -s "Hello" -B "Draft message"
  icloud email draft create -t "john@example.com" -s "Report" -f ./report.txt`,
	RunE: runDraftCreate,
}

var emailDraftListCmd = &cobra.Command{
	Use:   "list",
	Short: "List drafts",
	Long: `List all email drafts.

Examples:
  icloud email draft list
  icloud email draft list -n 50 -o json`,
	RunE: runDraftList,
}

var emailDraftUpdateCmd = &cobra.Command{
	Use:   "update <uid>",
	Short: "Update a draft",
	Long: `Update an existing draft by its UID.

Examples:
  icloud email draft update 12345 -s "New Subject"
  icloud email draft update 12345 -B "Updated body"`,
	Args: cobra.ExactArgs(1),
	RunE: runDraftUpdate,
}

var emailDraftDeleteCmd = &cobra.Command{
	Use:   "delete <uid>",
	Short: "Delete a draft",
	Long: `Delete a draft by its UID.

Examples:
  icloud email draft delete 12345
  icloud email draft delete 12345 -f`,
	Args: cobra.ExactArgs(1),
	RunE: runDraftDelete,
}

var emailDraftSendCmd = &cobra.Command{
	Use:   "send <uid>",
	Short: "Send a draft",
	Long: `Send an existing draft by its UID.

Examples:
  icloud email draft send 12345`,
	Args: cobra.ExactArgs(1),
	RunE: runDraftSend,
}

var (
	draftToFlag      string
	draftCCFlag      string
	draftSubjectFlag string
	draftBodyFlag    string
	draftLimitFlag   uint32
	draftForceFlag   bool
)

func init() {
	emailCmd.AddCommand(emailDraftCmd)
	emailDraftCmd.AddCommand(emailDraftCreateCmd)
	emailDraftCmd.AddCommand(emailDraftListCmd)
	emailDraftCmd.AddCommand(emailDraftUpdateCmd)
	emailDraftCmd.AddCommand(emailDraftDeleteCmd)
	emailDraftCmd.AddCommand(emailDraftSendCmd)

	emailDraftCreateCmd.Flags().StringVarP(&draftToFlag, "to", "t", "", "Recipient addresses (comma-separated)")
	emailDraftCreateCmd.Flags().StringVarP(&draftCCFlag, "cc", "c", "", "CC addresses (comma-separated)")
	emailDraftCreateCmd.Flags().StringVarP(&draftSubjectFlag, "subject", "s", "", "Email subject")
	emailDraftCreateCmd.Flags().StringVarP(&draftBodyFlag, "body", "B", "", "Email body text")
	emailDraftCreateCmd.MarkFlagRequired("to")
	emailDraftCreateCmd.MarkFlagRequired("subject")

	emailDraftListCmd.Flags().Uint32VarP(&draftLimitFlag, "limit", "n", 20, "Number of drafts to list")

	emailDraftUpdateCmd.Flags().StringVarP(&draftToFlag, "to", "t", "", "New recipient addresses")
	emailDraftUpdateCmd.Flags().StringVarP(&draftCCFlag, "cc", "c", "", "New CC addresses")
	emailDraftUpdateCmd.Flags().StringVarP(&draftSubjectFlag, "subject", "s", "", "New subject")
	emailDraftUpdateCmd.Flags().StringVarP(&draftBodyFlag, "body", "B", "", "New body text")

	emailDraftDeleteCmd.Flags().BoolVarP(&draftForceFlag, "force", "f", false, "Skip confirmation")
}

func runDraftCreate(cmd *cobra.Command, args []string) error {
	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	opts := email.DraftOptions{
		To:      parseAddresses(draftToFlag),
		CC:      parseAddresses(draftCCFlag),
		Subject: draftSubjectFlag,
		Body:    draftBodyFlag,
	}

	uid, err := client.CreateDraft(opts)
	if err != nil {
		return fmt.Errorf("failed to create draft: %w", err)
	}

	if uid > 0 {
		fmt.Printf("Draft created with UID: %d\n", uid)
	} else {
		fmt.Println("Draft created successfully")
	}
	return nil
}

func runDraftList(cmd *cobra.Command, args []string) error {
	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	drafts, err := client.ListDrafts(draftLimitFlag)
	if err != nil {
		return err
	}

	if emailOutputFlag == "json" {
		return outputEmailAsJSON(drafts)
	}

	return outputEmailList(drafts)
}

func runDraftUpdate(cmd *cobra.Command, args []string) error {
	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	existing, err := client.GetDraft(uid)
	if err != nil {
		return fmt.Errorf("failed to get existing draft: %w", err)
	}

	to := make([]string, len(existing.To))
	for i, addr := range existing.To {
		to[i] = addr.Address
	}
	cc := make([]string, len(existing.CC))
	for i, addr := range existing.CC {
		cc[i] = addr.Address
	}

	opts := email.DraftOptions{
		To:      to,
		CC:      cc,
		Subject: existing.Subject,
		Body:    existing.Body,
	}

	if draftToFlag != "" {
		opts.To = parseAddresses(draftToFlag)
	}
	if draftCCFlag != "" {
		opts.CC = parseAddresses(draftCCFlag)
	}
	if draftSubjectFlag != "" {
		opts.Subject = draftSubjectFlag
	}
	if draftBodyFlag != "" {
		opts.Body = draftBodyFlag
	}

	newUID, err := client.UpdateDraft(uid, opts)
	if err != nil {
		return fmt.Errorf("failed to update draft: %w", err)
	}

	if newUID > 0 {
		fmt.Printf("Draft updated, new UID: %d\n", newUID)
	} else {
		fmt.Println("Draft updated successfully")
	}
	return nil
}

func runDraftDelete(cmd *cobra.Command, args []string) error {
	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	if !draftForceFlag {
		fmt.Printf("Are you sure you want to delete draft %d? [y/N] ", uid)
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

	if err := client.DeleteDraft(uid); err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	fmt.Println("Draft deleted successfully")
	return nil
}

func runDraftSend(cmd *cobra.Command, args []string) error {
	uid, err := parseUID(args[0])
	if err != nil {
		return err
	}

	client, err := getEmailClient(emailAccountFlag)
	if err != nil {
		return err
	}

	if err := client.SendDraft(uid); err != nil {
		return fmt.Errorf("failed to send draft: %w", err)
	}

	fmt.Println("Draft sent successfully!")
	return nil
}
