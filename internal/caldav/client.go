package caldav

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

const (
	iCloudCalDAVURL = "https://caldav.icloud.com"
)

type Client struct {
	caldavClient *caldav.Client
	httpClient   webdav.HTTPClient
	email        string
	homeSet      string
}

func NewClient(email, password string) (*Client, error) {
	httpClient := webdav.HTTPClientWithBasicAuth(&http.Client{}, email, password)

	caldavClient, err := caldav.NewClient(httpClient, iCloudCalDAVURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	return &Client{
		caldavClient: caldavClient,
		httpClient:   httpClient,
		email:        email,
	}, nil
}

func (c *Client) FindCalendarHomeSet(ctx context.Context) (string, error) {
	if c.homeSet != "" {
		return c.homeSet, nil
	}

	principal, err := c.caldavClient.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to find current user principal: %w", err)
	}

	homeSet, err := c.caldavClient.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return "", fmt.Errorf("failed to find calendar home set: %w", err)
	}

	c.homeSet = homeSet
	return homeSet, nil
}
