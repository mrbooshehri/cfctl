package api

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

func (c *Client) ListDNSRecords(ctx context.Context, zoneID string) ([]cf.DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	records, _, err := c.api.ListDNSRecords(ctx, rc, cf.ListDNSRecordsParams{})
	return records, err
}

func (c *Client) CreateDNSRecord(ctx context.Context, zoneID string, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	return c.api.CreateDNSRecord(ctx, rc, params)
}

func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID string, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error) {
	rc := cf.ZoneIdentifier(zoneID)
	return c.api.UpdateDNSRecord(ctx, rc, params)
}

func (c *Client) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	rc := cf.ZoneIdentifier(zoneID)
	return c.api.DeleteDNSRecord(ctx, rc, recordID)
}
