package api

import (
	"context"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go"
)

type Client struct {
	api *cf.API
}

func New(token string) (*Client, error) {
	a, err := cf.NewWithAPIToken(token)
	if err != nil {
		return nil, fmt.Errorf("creating cloudflare client: %w", err)
	}
	return &Client{api: a}, nil
}

// Validate checks the token is usable by listing zones.
func (c *Client) Validate(ctx context.Context) error {
	_, err := c.api.ListZones(ctx)
	return err
}
