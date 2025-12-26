package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

const (
	IMAPServer = "imap.mail.me.com"
	IMAPPort   = 993
	SMTPServer = "smtp.mail.me.com"
	SMTPPort   = 587
)

type Client struct {
	email    string
	password string
}

func NewClient(email, password string) *Client {
	return &Client{
		email:    email,
		password: password,
	}
}

func (c *Client) imapUsername() string {
	if idx := strings.Index(c.email, "@"); idx > 0 {
		return c.email[:idx]
	}
	return c.email
}

func (c *Client) ConnectIMAP() (*imapclient.Client, error) {
	addr := fmt.Sprintf("%s:%d", IMAPServer, IMAPPort)

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: IMAPServer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	client := imapclient.New(conn, nil)

	if err := client.Login(c.imapUsername(), c.password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("IMAP authentication failed: %w", err)
	}

	return client, nil
}

func (c *Client) ConnectSMTP() (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClientStartTLS(conn, &tls.Config{ServerName: SMTPServer})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	auth := sasl.NewPlainClient("", c.email, c.password)
	if err := client.Auth(auth); err != nil {
		client.Close()
		return nil, fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return client, nil
}

func (c *Client) Email() string {
	return c.email
}
