package ebay

import (
	"context"
	"fmt"
	"strings"

	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

const (
	MsgReauthorizeAccountReadonly = "EBAY_REAUTHORIZATION_REQUIRED: complete eBay OAuth again to grant sell.account.readonly"
	MsgCompleteOAuth              = "unauthorized: complete eBay OAuth"
)

type Provider struct{}

func NewProvider() Provider       { return Provider{} }
func (Provider) Platform() string { return "ebay" }
func (Provider) Name() string     { return "eBay Sell API" }
func (Provider) Status() string   { return platformp.StatusBeta }
func (Provider) Capabilities() []platformp.Capability {
	return []platformp.Capability{platformp.CapProductPublish}
}
func (Provider) AuthSchema() platformp.AuthSchema {
	return platformp.AuthSchema{AuthType: "oauth2", Fields: []platformp.AuthField{}}
}
func (Provider) AppConfigSchema() platformp.PlatformAppConfigSchema {
	return platformp.EbayAppConfigSchema()
}
func (Provider) PublishConfigSchema() platformp.PlatformAppConfigSchema {
	return platformp.PublishConfigPresetForPlatform("ebay")
}

func (Provider) TestConnection(ctx context.Context, req platformp.TestConnectionRequest) (*platformp.TestConnectionResult, error) {
	cfg, err := ResolveRuntime(req)
	if err != nil {
		return &platformp.TestConnectionResult{OK: false, Message: err.Error()}, nil
	}
	if strings.TrimSpace(req.AccessToken) == "" {
		if strings.TrimSpace(req.RefreshToken) != "" {
			return &platformp.TestConnectionResult{OK: false, Message: MsgCompleteOAuth}, nil
		}
		return testApplicationCredentials(ctx, cfg)
	}
	res, _, err := ProbeUserConnection(ctx, cfg, req.AccessToken)
	return res, err
}

// ProbeUserConnection is Account API getPrivileges for a user token.
func ProbeUserConnection(ctx context.Context, cfg RuntimeConfig, accessToken string) (*platformp.TestConnectionResult, PrivilegeSnapshot, error) {
	return testUserPrivileges(ctx, cfg, accessToken)
}

func testApplicationCredentials(ctx context.Context, cfg RuntimeConfig) (*platformp.TestConnectionResult, error) {
	if _, err := ApplicationToken(ctx, cfg); err != nil {
		return &platformp.TestConnectionResult{OK: false, Message: err.Error()}, nil
	}
	region, currency := MarketplaceRegionCurrency(cfg.Marketplace, "")
	return &platformp.TestConnectionResult{
		OK:       true,
		Message:  fmt.Sprintf("eBay application token ok (%s, %s)", cfg.Environment, cfg.Marketplace),
		Region:   region,
		Currency: currency,
	}, nil
}

func testUserPrivileges(ctx context.Context, cfg RuntimeConfig, accessToken string) (*platformp.TestConnectionResult, PrivilegeSnapshot, error) {
	client := Client{Config: cfg}
	privileges, err := client.GetPrivileges(ctx, accessToken)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.Class == ErrorAuth {
			return &platformp.TestConnectionResult{OK: false, Message: MsgReauthorizeAccountReadonly + ": " + apiErr.Error()}, PrivilegeSnapshot{}, nil
		}
		return &platformp.TestConnectionResult{OK: false, Message: err.Error()}, PrivilegeSnapshot{}, nil
	}
	region, currency := MarketplaceRegionCurrency(cfg.Marketplace, privileges.SellingLimitCurrency)
	message := fmt.Sprintf("eBay Sell API ok (%s, %s)", cfg.Environment, cfg.Marketplace)
	if privileges.SellerRegistrationCompleted != nil && !*privileges.SellerRegistrationCompleted {
		message += "; sellerRegistrationCompleted=false"
	}
	return &platformp.TestConnectionResult{OK: true, Message: message, Region: region, Currency: currency}, privileges, nil
}

func RegisterProvider() { platformp.Register(NewProvider()) }
