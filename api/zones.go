package api

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

func (c *Client) ListZones(ctx context.Context) ([]cf.Zone, error) {
	return c.api.ListZones(ctx)
}
