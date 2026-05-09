package api

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

func (c *Client) ListAccessRules(ctx context.Context, zoneID string) ([]cf.AccessRule, error) {
	resp, err := c.api.ListZoneAccessRules(ctx, zoneID, cf.AccessRule{}, 1)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *Client) CreateAccessRule(ctx context.Context, zoneID string, rule cf.AccessRule) (*cf.AccessRuleResponse, error) {
	return c.api.CreateZoneAccessRule(ctx, zoneID, rule)
}

func (c *Client) DeleteAccessRule(ctx context.Context, zoneID, ruleID string) error {
	_, err := c.api.DeleteZoneAccessRule(ctx, zoneID, ruleID)
	return err
}
