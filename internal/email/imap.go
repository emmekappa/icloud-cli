package email

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func (c *Client) ListMailboxes() ([]Mailbox, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return nil, err
	}
	defer imapClient.Close()

	listCmd := imapClient.List("", "*", nil)
	mailboxes := []Mailbox{}

	for {
		data := listCmd.Next()
		if data == nil {
			break
		}

		attrs := make([]string, 0)
		for _, a := range data.Attrs {
			attrs = append(attrs, string(a))
		}

		mailbox := Mailbox{
			Name:       data.Mailbox,
			Delimiter:  string(data.Delim),
			Attributes: strings.Join(attrs, ", "),
		}
		mailboxes = append(mailboxes, mailbox)
	}

	if err := listCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close list command: %w", err)
	}

	return mailboxes, nil
}

func (c *Client) ListEmails(mailbox string, limit uint32) ([]Email, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return nil, err
	}
	defer imapClient.Close()

	selectCmd := imapClient.Select(mailbox, nil)
	selectData, err := selectCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	if selectData.NumMessages == 0 {
		return []Email{}, nil
	}

	start := uint32(1)
	if selectData.NumMessages > limit {
		start = selectData.NumMessages - limit + 1
	}

	var seqSet imap.SeqSet
	seqSet.AddRange(start, selectData.NumMessages)

	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Flags:       true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{},
	}

	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	emails := []Email{}

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		email := c.parseMessage(msg, mailbox)
		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fetch command: %w", err)
	}

	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

func (c *Client) GetEmail(mailbox string, uid uint32) (*Email, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return nil, err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	fetchOptions := &imap.FetchOptions{
		UID:      true,
		Flags:    true,
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	fetchCmd := imapClient.Fetch(uidSet, fetchOptions)
	msg := fetchCmd.Next()
	if msg == nil {
		if err := fetchCmd.Close(); err != nil {
			return nil, fmt.Errorf("failed to close fetch command: %w", err)
		}
		return nil, fmt.Errorf("email with UID %d not found in %s", uid, mailbox)
	}

	email := c.parseMessageWithBody(msg, mailbox)

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fetch command: %w", err)
	}

	return &email, nil
}

func (c *Client) SearchEmails(mailbox string, criteria *imap.SearchCriteria, limit uint32) ([]Email, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return nil, err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	searchCmd := imapClient.Search(criteria, nil)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchData.AllSeqNums()) == 0 {
		return []Email{}, nil
	}

	seqNums := searchData.AllSeqNums()
	if uint32(len(seqNums)) > limit {
		seqNums = seqNums[len(seqNums)-int(limit):]
	}

	var seqSet imap.SeqSet
	for _, n := range seqNums {
		seqSet.AddNum(n)
	}

	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Flags:       true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{},
	}

	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	emails := []Email{}

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		email := c.parseMessage(msg, mailbox)
		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fetch command: %w", err)
	}

	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

func (c *Client) MoveEmail(sourceMailbox string, uid uint32, destMailbox string) error {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(sourceMailbox, nil).Wait(); err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", sourceMailbox, err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	copyCmd := imapClient.Copy(uidSet, destMailbox)
	if _, err := copyCmd.Wait(); err != nil {
		return fmt.Errorf("failed to copy email to %s: %w", destMailbox, err)
	}

	storeCmd := imapClient.Store(uidSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to mark email for deletion: %w", err)
	}

	if err := imapClient.Expunge().Close(); err != nil {
		return fmt.Errorf("failed to expunge: %w", err)
	}

	return nil
}

func (c *Client) DeleteEmail(mailbox string, uid uint32) error {
	return c.MoveEmail(mailbox, uid, "Trash")
}

func (c *Client) MarkEmail(mailbox string, uid uint32, seen bool) error {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(mailbox, nil).Wait(); err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	op := imap.StoreFlagsAdd
	if !seen {
		op = imap.StoreFlagsDel
	}

	storeCmd := imapClient.Store(uidSet, &imap.StoreFlags{
		Op:    op,
		Flags: []imap.Flag{imap.FlagSeen},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to update seen flag: %w", err)
	}

	return nil
}

func (c *Client) FlagEmail(mailbox string, uid uint32, flagged bool) error {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(mailbox, nil).Wait(); err != nil {
		return fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(imap.UID(uid))

	op := imap.StoreFlagsAdd
	if !flagged {
		op = imap.StoreFlagsDel
	}

	storeCmd := imapClient.Store(uidSet, &imap.StoreFlags{
		Op:    op,
		Flags: []imap.Flag{imap.FlagFlagged},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to update flagged flag: %w", err)
	}

	return nil
}

func (c *Client) parseMessage(msg *imapclient.FetchMessageData, mailbox string) Email {
	email := Email{Mailbox: mailbox}

	for {
		item := msg.Next()
		if item == nil {
			break
		}

		switch data := item.(type) {
		case imapclient.FetchItemDataUID:
			email.UID = uint32(data.UID)
		case imapclient.FetchItemDataFlags:
			for _, flag := range data.Flags {
				switch flag {
				case imap.FlagSeen:
					email.Seen = true
				case imap.FlagFlagged:
					email.Flagged = true
				case imap.FlagAnswered:
					email.Answered = true
				}
			}
		case imapclient.FetchItemDataEnvelope:
			email.MessageID = data.Envelope.MessageID
			email.Subject = data.Envelope.Subject
			email.Date = data.Envelope.Date
			if len(data.Envelope.InReplyTo) > 0 {
				email.InReplyTo = data.Envelope.InReplyTo[0]
			}
			email.From = convertAddresses(data.Envelope.From)
			email.To = convertAddresses(data.Envelope.To)
			email.CC = convertAddresses(data.Envelope.Cc)
			email.BCC = convertAddresses(data.Envelope.Bcc)
		}
	}

	return email
}

func (c *Client) parseMessageWithBody(msg *imapclient.FetchMessageData, mailbox string) Email {
	email := Email{Mailbox: mailbox}

	for {
		item := msg.Next()
		if item == nil {
			break
		}

		switch data := item.(type) {
		case imapclient.FetchItemDataUID:
			email.UID = uint32(data.UID)
		case imapclient.FetchItemDataFlags:
			for _, flag := range data.Flags {
				switch flag {
				case imap.FlagSeen:
					email.Seen = true
				case imap.FlagFlagged:
					email.Flagged = true
				case imap.FlagAnswered:
					email.Answered = true
				}
			}
		case imapclient.FetchItemDataEnvelope:
			email.MessageID = data.Envelope.MessageID
			email.Subject = data.Envelope.Subject
			email.Date = data.Envelope.Date
			if len(data.Envelope.InReplyTo) > 0 {
				email.InReplyTo = data.Envelope.InReplyTo[0]
			}
			email.From = convertAddresses(data.Envelope.From)
			email.To = convertAddresses(data.Envelope.To)
			email.CC = convertAddresses(data.Envelope.Cc)
			email.BCC = convertAddresses(data.Envelope.Bcc)
		case imapclient.FetchItemDataBodySection:
			body, _ := io.ReadAll(data.Literal)
			email.Body = string(body)
		}
	}

	return email
}

func convertAddresses(addrs []imap.Address) []Address {
	result := make([]Address, 0, len(addrs))
	for _, a := range addrs {
		addr := Address{
			Name:    a.Name,
			Address: a.Addr(),
		}
		result = append(result, addr)
	}
	return result
}

func ParseSearchQuery(query string) (*imap.SearchCriteria, error) {
	criteria := &imap.SearchCriteria{}
	parts := strings.Fields(query)

	for _, part := range parts {
		switch {
		case strings.HasPrefix(strings.ToUpper(part), "FROM:"):
			criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
				Key:   "From",
				Value: strings.TrimPrefix(part, "FROM:"),
			})
		case strings.HasPrefix(strings.ToUpper(part), "TO:"):
			criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
				Key:   "To",
				Value: strings.TrimPrefix(part, "TO:"),
			})
		case strings.HasPrefix(strings.ToUpper(part), "SUBJECT:"):
			criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
				Key:   "Subject",
				Value: strings.TrimPrefix(part, "SUBJECT:"),
			})
		case strings.HasPrefix(strings.ToUpper(part), "BODY:"):
			criteria.Body = append(criteria.Body, strings.TrimPrefix(part, "BODY:"))
		case strings.HasPrefix(strings.ToUpper(part), "SINCE:"):
			dateStr := strings.TrimPrefix(part, "SINCE:")
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid date format for SINCE: %s (use YYYY-MM-DD)", dateStr)
			}
			criteria.Since = t
		case strings.HasPrefix(strings.ToUpper(part), "BEFORE:"):
			dateStr := strings.TrimPrefix(part, "BEFORE:")
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return nil, fmt.Errorf("invalid date format for BEFORE: %s (use YYYY-MM-DD)", dateStr)
			}
			criteria.Before = t
		case strings.ToUpper(part) == "UNSEEN":
			criteria.NotFlag = append(criteria.NotFlag, imap.FlagSeen)
		case strings.ToUpper(part) == "FLAGGED":
			criteria.Flag = append(criteria.Flag, imap.FlagFlagged)
		default:
			criteria.Body = append(criteria.Body, part)
		}
	}

	return criteria, nil
}

func (c *Client) GetEmailByMessageID(mailbox, messageID string) (*Email, error) {
	imapClient, err := c.ConnectIMAP()
	if err != nil {
		return nil, err
	}
	defer imapClient.Close()

	if _, err := imapClient.Select(mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Message-ID", Value: messageID},
		},
	}

	searchCmd := imapClient.Search(criteria, nil)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	seqNums := searchData.AllSeqNums()
	if len(seqNums) == 0 {
		return nil, fmt.Errorf("email with Message-ID %s not found in %s", messageID, mailbox)
	}

	var seqSet imap.SeqSet
	seqSet.AddNum(seqNums[0])

	fetchOptions := &imap.FetchOptions{
		UID:      true,
		Flags:    true,
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	fetchCmd := imapClient.Fetch(seqSet, fetchOptions)
	msg := fetchCmd.Next()
	if msg == nil {
		if err := fetchCmd.Close(); err != nil {
			return nil, fmt.Errorf("failed to close fetch command: %w", err)
		}
		return nil, fmt.Errorf("email with Message-ID %s not found", messageID)
	}

	email := c.parseMessageWithBody(msg, mailbox)

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fetch command: %w", err)
	}

	return &email, nil
}

func BuildReplyEmail(original *Email, replyAll bool, senderEmail string) *Email {
	reply := &Email{
		To:         []Address{original.From[0]},
		InReplyTo:  original.MessageID,
		References: append(original.References, original.MessageID),
	}

	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}
	reply.Subject = subject

	if replyAll {
		for _, addr := range original.To {
			if addr.Address != senderEmail {
				reply.To = append(reply.To, addr)
			}
		}
		for _, addr := range original.CC {
			if addr.Address != senderEmail {
				reply.CC = append(reply.CC, addr)
			}
		}
	}

	return reply
}
