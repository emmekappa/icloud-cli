package email

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
)

const DraftsMailbox = "Drafts"

type DraftOptions struct {
	To       []string
	CC       []string
	Subject  string
	Body     string
	HTMLBody string
}

func (c *Client) CreateDraft(opts DraftOptions) (uint32, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return 0, err
	}
	defer imapClient.Close()

	msg := c.buildDraftMessage(opts)

	appendCmd := imapClient.Append(DraftsMailbox, int64(len(msg)), nil)
	if _, err := appendCmd.Write(msg); err != nil {
		return 0, fmt.Errorf("failed to write draft message: %w", err)
	}

	if err := appendCmd.Close(); err != nil {
		return 0, fmt.Errorf("failed to append draft: %w", err)
	}

	return 0, nil
}

func (c *Client) ListDrafts(limit uint32) ([]Email, error) {
	return c.ListEmails(DraftsMailbox, limit)
}

func (c *Client) GetDraft(uid uint32) (*Email, error) {
	return c.GetEmail(DraftsMailbox, uid)
}

func (c *Client) UpdateDraft(uid uint32, opts DraftOptions) (uint32, error) {
	if err := c.deleteDraftPermanently(uid); err != nil {
		return 0, err
	}

	return c.CreateDraft(opts)
}

func (c *Client) DeleteDraft(uid uint32) error {
	return c.deleteDraftPermanently(uid)
}

func (c *Client) deleteDraftPermanently(uid uint32) error {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(DraftsMailbox, nil).Wait(); err != nil {
		return fmt.Errorf("failed to select Drafts mailbox: %w", err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	storeCmd := imapClient.Store(uidSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to mark draft for deletion: %w", err)
	}

	if err := imapClient.Expunge().Close(); err != nil {
		return fmt.Errorf("failed to expunge: %w", err)
	}

	return nil
}

func (c *Client) SendDraft(uid uint32) error {
	draft, err := c.GetDraft(uid)
	if err != nil {
		return fmt.Errorf("failed to get draft: %w", err)
	}

	to := make([]string, len(draft.To))
	for i, addr := range draft.To {
		to[i] = addr.Address
	}

	cc := make([]string, len(draft.CC))
	for i, addr := range draft.CC {
		cc[i] = addr.Address
	}

	sendOpts := SendOptions{
		To:      to,
		CC:      cc,
		Subject: draft.Subject,
		Body:    draft.Body,
	}

	if err := c.SendEmail(sendOpts); err != nil {
		return fmt.Errorf("failed to send draft: %w", err)
	}

	if err := c.deleteDraftPermanently(uid); err != nil {
		return fmt.Errorf("email sent but failed to delete draft: %w", err)
	}

	return nil
}

func (c *Client) buildDraftMessage(opts DraftOptions) []byte {
	var buf bytes.Buffer

	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), strings.Split(c.email, "@")[1])
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("From: %s\r\n", c.email))

	if len(opts.To) > 0 {
		buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(opts.To, ", ")))
	}
	if len(opts.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(opts.CC, ", ")))
	}
	if opts.Subject != "" {
		buf.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("utf-8", opts.Subject)))
	}

	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("X-Uniform-Type-Identifier: com.apple.mail-draft\r\n")

	if opts.HTMLBody != "" {
		boundary := fmt.Sprintf("----=_Part_%s", uuid.New().String())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(opts.Body)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(opts.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(opts.Body)
	}

	return buf.Bytes()
}
