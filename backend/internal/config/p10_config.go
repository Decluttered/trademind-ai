package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const ErrCodeP10BoundaryViolation = "P10_BOUNDARY_VIOLATION"

// P10Config describes repository-side controls. This release intentionally supports only L0 at runtime.
type P10Config struct {
	CurrentAllowedLevel        string
	OfflineOAuthEnabled        bool
	LocalCredentialKey         string
	LocalCredentialKeyRef      string
	OAuthStateTTL              time.Duration
	RedirectAllowlist          []string
	DouyinAPIBaseURL           string
	ProviderRequestTimeout     time.Duration
	ProviderConnectTimeout     time.Duration
	ProviderResponseHeaderTime time.Duration
	ProviderMaxResponseBytes   int64
	ProviderConcurrency        int
	SKUPageSize                int
	PaginationLimit            int
	RealProviderEnabled        bool
	RealPlatformNetworkEnabled bool
	RealCredentialsEnabled     bool
	RealInventoryReadEnabled   bool
	InventoryMutationEnabled   bool
	BackgroundWorkerEnabled    bool
	AutomaticRetryEnabled      bool
}

func loadP10Config(appEnv string) P10Config {
	_ = appEnv
	return P10Config{
		CurrentAllowedLevel:        strings.ToUpper(strings.TrimSpace(firstNonEmpty(os.Getenv("P10_CURRENT_ALLOWED_LEVEL"), "L0"))),
		OfflineOAuthEnabled:        envBool(os.Getenv("P10_OFFLINE_OAUTH_ENABLED"), false),
		LocalCredentialKey:         strings.TrimSpace(os.Getenv("P10_LOCAL_CREDENTIAL_KEY")),
		LocalCredentialKeyRef:      strings.TrimSpace(firstNonEmpty(os.Getenv("P10_LOCAL_CREDENTIAL_KEY_REF"), "local-development-v1")),
		OAuthStateTTL:              time.Duration(atoiOrDefault(os.Getenv("P10_OAUTH_STATE_TTL_SECONDS"), 600)) * time.Second,
		RedirectAllowlist:          splitCSV(os.Getenv("P10_OAUTH_REDIRECT_ALLOWLIST")),
		DouyinAPIBaseURL:           strings.TrimRight(strings.TrimSpace(os.Getenv("P10_DOUYIN_API_BASE_URL")), "/"),
		ProviderRequestTimeout:     time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_REQUEST_TIMEOUT_SECONDS"), 30)) * time.Second,
		ProviderConnectTimeout:     time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_CONNECT_TIMEOUT_SECONDS"), 5)) * time.Second,
		ProviderResponseHeaderTime: time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_RESPONSE_HEADER_TIMEOUT_SECONDS"), 15)) * time.Second,
		ProviderMaxResponseBytes:   int64(atoiOrDefault(os.Getenv("P10_PROVIDER_MAX_RESPONSE_BYTES"), 2*1024*1024)),
		ProviderConcurrency:        atoiOrDefault(os.Getenv("P10_PROVIDER_CONCURRENCY"), 2),
		SKUPageSize:                atoiOrDefault(os.Getenv("P10_SKU_PAGE_SIZE"), 50),
		PaginationLimit:            atoiOrDefault(os.Getenv("P10_PAGINATION_LIMIT"), 100),
		RealProviderEnabled:        envBool(os.Getenv("P10_REAL_PROVIDER_ENABLED"), false),
		RealPlatformNetworkEnabled: envBool(os.Getenv("P10_REAL_PLATFORM_NETWORK_ENABLED"), false),
		RealCredentialsEnabled:     envBool(os.Getenv("P10_REAL_CREDENTIALS_ENABLED"), false),
		RealInventoryReadEnabled:   envBool(os.Getenv("P10_REAL_INVENTORY_READ_ENABLED"), false),
		InventoryMutationEnabled:   envBool(os.Getenv("P10_INVENTORY_MUTATION_ENABLED"), false),
		BackgroundWorkerEnabled:    envBool(os.Getenv("P10_BACKGROUND_WORKER_ENABLED"), false),
		AutomaticRetryEnabled:      envBool(os.Getenv("P10_AUTOMATIC_RETRY_ENABLED"), false),
	}
}

func (c P10Config) Validate(appEnv string) error {
	if c.CurrentAllowedLevel != "L0" {
		return fmt.Errorf("%s: only L0 is allowed before manual and external acceptance", ErrCodeP10BoundaryViolation)
	}
	if c.RealProviderEnabled || c.RealPlatformNetworkEnabled || c.RealCredentialsEnabled || c.RealInventoryReadEnabled || c.InventoryMutationEnabled || c.BackgroundWorkerEnabled || c.AutomaticRetryEnabled {
		return fmt.Errorf("%s: all real, mutation, worker, and retry capabilities must remain disabled at L0", ErrCodeP10BoundaryViolation)
	}
	if c.OfflineOAuthEnabled && IsStagingOrProduction(appEnv) {
		return fmt.Errorf("%s: offline OAuth is development/test only", ErrCodeP10BoundaryViolation)
	}
	if c.LocalCredentialKey != "" && IsStagingOrProduction(appEnv) {
		return fmt.Errorf("%s: local credential keys are forbidden in staging/production", ErrCodeP10BoundaryViolation)
	}
	if c.OAuthStateTTL < time.Minute || c.OAuthStateTTL > 30*time.Minute {
		return fmt.Errorf("%s: OAuth state TTL must be between 60 and 1800 seconds", ErrCodeP10BoundaryViolation)
	}
	if c.ProviderRequestTimeout < time.Second || c.ProviderRequestTimeout > 2*time.Minute || c.ProviderConnectTimeout <= 0 || c.ProviderResponseHeaderTime <= 0 {
		return fmt.Errorf("%s: provider timeout configuration is invalid", ErrCodeP10BoundaryViolation)
	}
	if c.ProviderMaxResponseBytes < 64*1024 || c.ProviderMaxResponseBytes > 10*1024*1024 || c.ProviderConcurrency < 1 || c.ProviderConcurrency > 10 || c.SKUPageSize < 1 || c.SKUPageSize > 100 || c.PaginationLimit < 1 || c.PaginationLimit > 100 {
		return fmt.Errorf("%s: provider capacity configuration exceeds the approved limits", ErrCodeP10BoundaryViolation)
	}
	if c.DouyinAPIBaseURL != "" {
		u, err := url.Parse(c.DouyinAPIBaseURL)
		if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "openapi-fxg.jinritemai.com") || (u.Port() != "" && u.Port() != "443") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("%s: P10_DOUYIN_API_BASE_URL must be a trusted HTTPS URL without userinfo", ErrCodeP10BoundaryViolation)
		}
	}
	for _, raw := range c.RedirectAllowlist {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Scheme == "" || u.Host == "" || u.Fragment != "" {
			return fmt.Errorf("%s: invalid OAuth redirect allowlist entry", ErrCodeP10BoundaryViolation)
		}
	}
	return nil
}

func parseTenantIDs(raw string) []int64 {
	parts := splitCSV(raw)
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		if value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && value > 0 {
			out = append(out, value)
		}
	}
	return out
}
