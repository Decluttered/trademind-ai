package ebay

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
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
		return &platformp.TestConnectionResult{OK: false, Message: "unauthorized: complete eBay OAuth"}, nil
	}
	client := Client{Config: cfg}
	_, err = client.call(ctx, req.AccessToken, http.MethodGet, "/sell/account/v1/privilege", nil)
	if err != nil {
		return &platformp.TestConnectionResult{OK: false, Message: err.Error()}, nil
	}
	return &platformp.TestConnectionResult{OK: true, Message: fmt.Sprintf("eBay Sell API ok (%s, %s)", cfg.Environment, cfg.Marketplace), Region: "DE", Currency: "EUR"}, nil
}

func RegisterProvider() { platformp.Register(NewProvider()) }
