package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

type SendOptions struct {
	To       []string
	CC       []string
	BCC      []string
	Subject  string
	Body     string
	HTMLBody string
	BodyFile string

	InReplyTo  string
	References []string
}

func (c *Client) SendEmail(opts SendOptions) error {
	smtpClient, err := c.ConnectSMTP()
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	if err := smtpClient.Mail(c.email, nil); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	allRecipients := append(append(opts.To, opts.CC...), opts.BCC...)
	for _, rcpt := range allRecipients {
		if err := smtpClient.Rcpt(rcpt, nil); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", rcpt, err)
		}
	}

	msg, err := c.buildMessage(opts)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	wc, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := wc.Write(msg); err != nil {
		wc.Close()
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

func (c *Client) buildMessage(opts SendOptions) ([]byte, error) {
	var buf bytes.Buffer

	messageID := fmt.Sprintf("<%s@%s>", uuid.New().String(), strings.Split(c.email, "@")[1])
	buf.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString(fmt.Sprintf("From: %s\r\n", c.email))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(opts.To, ", ")))
	if len(opts.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(opts.CC, ", ")))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(opts.Subject)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if opts.InReplyTo != "" {
		buf.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", opts.InReplyTo))
	}
	if len(opts.References) > 0 {
		buf.WriteString(fmt.Sprintf("References: %s\r\n", strings.Join(opts.References, " ")))
	}

	body := opts.Body
	if opts.BodyFile != "" {
		content, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read body file: %w", err)
		}
		body = string(content)
		if strings.HasSuffix(strings.ToLower(opts.BodyFile), ".html") {
			opts.HTMLBody = body
			body = ""
		}
	}

	if opts.HTMLBody != "" && body != "" {
		boundary := fmt.Sprintf("----=_Part_%s", uuid.New().String())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(body)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(opts.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if opts.HTMLBody != "" {
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(opts.HTMLBody)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(body)
	}

	return buf.Bytes(), nil
}

func encodeSubject(subject string) string {
	return mime.QEncoding.Encode("utf-8", subject)
}

var _ = multipart.Writer{}
var _ = textproto.MIMEHeader{}
var _ = filepath.Base
var _ = io.Discard
var _ smtp.Client
