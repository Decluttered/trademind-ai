package ebay

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

const InventoryScope = "https://api.ebay.com/oauth/api_scope/sell.inventory"

var AuthorizationScopes = []string{
	InventoryScope,
	"https://api.ebay.com/oauth/api_scope/sell.fulfillment",
	"https://api.ebay.com/oauth/api_scope/sell.finances",
	"https://api.ebay.com/oauth/api_scope/commerce.media",
}

type RuntimeConfig struct {
	Environment  string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
	Marketplace  string
	Timeout      time.Duration
}

func ResolveRuntime(req platformp.TestConnectionRequest) (RuntimeConfig, error) {
	extra := req.Extra
	get := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(extra[key]); value != "" {
				return value
			}
		}
		return ""
	}
	env := strings.ToLower(get("environment", "ebay_env"))
	if env == "" {
		env = "sandbox"
	}
	if env != "sandbox" && env != "production" {
		return RuntimeConfig{}, fmt.Errorf("EBAY_ENV must be sandbox or production")
	}
	apiBase := get("api_base_url")
	authBase := get("auth_base_url")
	if apiBase == "" {
		if env == "sandbox" {
			apiBase = "https://api.sandbox.ebay.com"
		} else {
			apiBase = "https://api.ebay.com"
		}
	}
	if authBase == "" {
		if env == "sandbox" {
			authBase = "https://auth.sandbox.ebay.com"
		} else {
			authBase = "https://auth.ebay.com"
		}
	}
	timeout := 30 * time.Second
	if raw := get("timeout_sec"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 1 || seconds > 120 {
			return RuntimeConfig{}, fmt.Errorf("invalid eBay timeout_sec")
		}
		timeout = time.Duration(seconds) * time.Second
	}
	marketplace := strings.TrimSpace(req.MarketplaceID)
	if marketplace == "" {
		marketplace = get("marketplace_id")
	}
	if marketplace == "" {
		marketplace = "EBAY_DE"
	}
	return RuntimeConfig{Environment: env, ClientID: first(req.AppKey, get("client_id")), ClientSecret: first(req.AppSecret, get("client_secret")), RedirectURI: get("redirect_uri"), APIBaseURL: strings.TrimRight(apiBase, "/"), AuthBaseURL: strings.TrimRight(authBase, "/"), Marketplace: marketplace, Timeout: timeout}, nil
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
