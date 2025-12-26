package email

import (
	"time"
)

type Email struct {
	MessageID   string       `json:"message_id"`
	UID         uint32       `json:"uid"`
	Mailbox     string       `json:"mailbox"`
	From        []Address    `json:"from"`
	To          []Address    `json:"to"`
	CC          []Address    `json:"cc,omitempty"`
	BCC         []Address    `json:"bcc,omitempty"`
	Subject     string       `json:"subject"`
	Date        time.Time    `json:"date"`
	Body        string       `json:"body,omitempty"`
	HTMLBody    string       `json:"html_body,omitempty"`
	Seen        bool         `json:"seen"`
	Flagged     bool         `json:"flagged"`
	Answered    bool         `json:"answered"`
	InReplyTo   string       `json:"in_reply_to,omitempty"`
	References  []string     `json:"references,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

func (a Address) String() string {
	if a.Name != "" {
		return a.Name + " <" + a.Address + ">"
	}
	return a.Address
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type Mailbox struct {
	Name       string `json:"name"`
	Delimiter  string `json:"delimiter"`
	Messages   uint32 `json:"messages"`
	Unseen     uint32 `json:"unseen"`
	Attributes string `json:"attributes,omitempty"`
}
