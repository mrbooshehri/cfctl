package api

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

func (c *Client) ListCertificatePacks(ctx context.Context, zoneID string) ([]cf.CertificatePack, error) {
	return c.api.ListCertificatePacks(ctx, zoneID)
}
