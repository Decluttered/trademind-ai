package ebay

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

// Official OAuth scope URIs: https://developer.ebay.com/api-docs/static/oauth-scopes.html
const (
	InventoryScope        = "https://api.ebay.com/oauth/api_scope/sell.inventory"
	AccountReadonlyScope  = "https://api.ebay.com/oauth/api_scope/sell.account.readonly"
	applicationTokenScope = "https://api.ebay.com/oauth/api_scope"
)

// AuthorizationScopes is the consent list for authorization-code grant.
// sell.account.readonly is required for Account API getPrivileges (connection probe).
var AuthorizationScopes = []string{
	InventoryScope,
	"https://api.ebay.com/oauth/api_scope/sell.fulfillment",
	"https://api.ebay.com/oauth/api_scope/sell.finances",
	"https://api.ebay.com/oauth/api_scope/commerce.media",
	AccountReadonlyScope,
}

// legacyRefreshScopes is used when a stored shop token has no scopes JSON.
// It omits sell.account.readonly so existing refresh tokens are not invalidated.
var legacyRefreshScopes = []string{
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
		switch strings.ToLower(get("sandbox_enabled")) {
		case "true":
			env = "sandbox"
		case "false":
			env = "production"
		default:
			env = "sandbox"
		}
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
		return RuntimeConfig{}, fmt.Errorf("marketplace_id is required")
	}
	ruName := get("redirect_uri")
	if ruName != "" {
		if err := ValidateRuName(ruName); err != nil {
			return RuntimeConfig{}, err
		}
	}
	return RuntimeConfig{
		Environment:  env,
		ClientID:     first(req.AppKey, get("client_id")),
		ClientSecret: first(req.AppSecret, get("client_secret")),
		RedirectURI:  ruName,
		APIBaseURL:   strings.TrimRight(apiBase, "/"),
		AuthBaseURL:  strings.TrimRight(authBase, "/"),
		Marketplace:  marketplace,
		Timeout:      timeout,
	}, nil
}

// RuntimeFromMergedMap validates settings.platform_ebay (lowercase keys).
func RuntimeFromMergedMap(m map[string]string) (RuntimeConfig, error) {
	extra := map[string]string{}
	for key, value := range m {
		extra[strings.TrimSpace(strings.ToLower(key))] = strings.TrimSpace(value)
	}
	cfg, err := ResolveRuntime(platformp.TestConnectionRequest{
		AppKey:        extra["client_id"],
		AppSecret:     extra["client_secret"],
		MarketplaceID: extra["marketplace_id"],
		Extra:         extra,
	})
	if err != nil {
		return RuntimeConfig{}, err
	}
	if err := ValidateRuName(cfg.RedirectURI); err != nil {
		return RuntimeConfig{}, err
	}
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return RuntimeConfig{}, fmt.Errorf("eBay client_id and client_secret are required")
	}
	return cfg, nil
}

// ValidateRuName rejects HTTPS callback URLs. eBay OAuth redirect_uri is the
// RuName from Developer Portal (OAuth Redirect URL name), not the https URL
// registered behind that name.
func ValidateRuName(value string) error {
	ruName := strings.TrimSpace(value)
	if ruName == "" {
		return fmt.Errorf("eBay RuName (redirect_uri) is required")
	}
	lower := strings.ToLower(ruName)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("eBay redirect_uri must be the RuName from Developer Portal, not an https URL")
	}
	if strings.ContainsAny(ruName, " \t\n\r") {
		return fmt.Errorf("eBay RuName must not contain whitespace")
	}
	return nil
}

func RefreshScopes(granted []string) []string {
	if len(granted) == 0 {
		return append([]string{}, legacyRefreshScopes...)
	}
	out := make([]string, 0, len(granted))
	for _, scope := range granted {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return append([]string{}, legacyRefreshScopes...)
	}
	return out
}

func HasAccountReadonlyScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.TrimSpace(scope) == AccountReadonlyScope {
			return true
		}
	}
	return false
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
