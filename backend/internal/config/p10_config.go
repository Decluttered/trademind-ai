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

// P10Config describes fail-closed production capability controls.
type P10Config struct {
	CurrentAllowedLevel          string
	OfflineOAuthEnabled          bool
	LocalCredentialKey           string
	LocalCredentialKeyRef        string
	OAuthStateTTL                time.Duration
	RedirectAllowlist            []string
	DouyinAPIBaseURL             string
	ProviderRequestTimeout       time.Duration
	ProviderConnectTimeout       time.Duration
	ProviderResponseHeaderTime   time.Duration
	ProviderMaxResponseBytes     int64
	ProviderConcurrency          int
	SKUPageSize                  int
	PaginationLimit              int
	RealProviderEnabled          bool
	RealPlatformNetworkEnabled   bool
	RealCredentialsEnabled       bool
	RealInventoryReadEnabled     bool
	RealProductDraftWriteEnabled bool
	InventoryMutationEnabled     bool
	BackgroundWorkerEnabled      bool
	AutomaticRetryEnabled        bool
}

func loadP10Config(appEnv string) P10Config {
	_ = appEnv
	return P10Config{
		CurrentAllowedLevel:          strings.ToUpper(strings.TrimSpace(firstNonEmpty(os.Getenv("P10_CURRENT_ALLOWED_LEVEL"), "L0"))),
		OfflineOAuthEnabled:          envBool(os.Getenv("P10_OFFLINE_OAUTH_ENABLED"), false),
		LocalCredentialKey:           strings.TrimSpace(os.Getenv("P10_LOCAL_CREDENTIAL_KEY")),
		LocalCredentialKeyRef:        strings.TrimSpace(firstNonEmpty(os.Getenv("P10_LOCAL_CREDENTIAL_KEY_REF"), "local-development-v1")),
		OAuthStateTTL:                time.Duration(atoiOrDefault(os.Getenv("P10_OAUTH_STATE_TTL_SECONDS"), 600)) * time.Second,
		RedirectAllowlist:            splitCSV(os.Getenv("P10_OAUTH_REDIRECT_ALLOWLIST")),
		DouyinAPIBaseURL:             strings.TrimRight(strings.TrimSpace(os.Getenv("P10_DOUYIN_API_BASE_URL")), "/"),
		ProviderRequestTimeout:       time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_REQUEST_TIMEOUT_SECONDS"), 30)) * time.Second,
		ProviderConnectTimeout:       time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_CONNECT_TIMEOUT_SECONDS"), 5)) * time.Second,
		ProviderResponseHeaderTime:   time.Duration(atoiOrDefault(os.Getenv("P10_PROVIDER_RESPONSE_HEADER_TIMEOUT_SECONDS"), 15)) * time.Second,
		ProviderMaxResponseBytes:     int64(atoiOrDefault(os.Getenv("P10_PROVIDER_MAX_RESPONSE_BYTES"), 2*1024*1024)),
		ProviderConcurrency:          atoiOrDefault(os.Getenv("P10_PROVIDER_CONCURRENCY"), 2),
		SKUPageSize:                  atoiOrDefault(os.Getenv("P10_SKU_PAGE_SIZE"), 50),
		PaginationLimit:              atoiOrDefault(os.Getenv("P10_PAGINATION_LIMIT"), 100),
		RealProviderEnabled:          envBool(os.Getenv("P10_REAL_PROVIDER_ENABLED"), false),
		RealPlatformNetworkEnabled:   envBool(os.Getenv("P10_REAL_PLATFORM_NETWORK_ENABLED"), false),
		RealCredentialsEnabled:       envBool(os.Getenv("P10_REAL_CREDENTIALS_ENABLED"), false),
		RealInventoryReadEnabled:     envBool(os.Getenv("P10_REAL_INVENTORY_READ_ENABLED"), false),
		RealProductDraftWriteEnabled: envBool(os.Getenv("P10_REAL_PRODUCT_DRAFT_WRITE_ENABLED"), false),
		InventoryMutationEnabled:     envBool(os.Getenv("P10_INVENTORY_MUTATION_ENABLED"), false),
		BackgroundWorkerEnabled:      envBool(os.Getenv("P10_BACKGROUND_WORKER_ENABLED"), false),
		AutomaticRetryEnabled:        envBool(os.Getenv("P10_AUTOMATIC_RETRY_ENABLED"), false),
	}
}

func (c P10Config) Validate(appEnv string) error {
	level := strings.ToUpper(strings.TrimSpace(c.CurrentAllowedLevel))
	if level == "" {
		level = "L0"
	}
	if level != "L0" && level != "L1" && level != "L3" {
		return fmt.Errorf("%s: allowed level must be L0, L1, or L3", ErrCodeP10BoundaryViolation)
	}
	anyRealCapability := c.RealProviderEnabled || c.RealPlatformNetworkEnabled || c.RealCredentialsEnabled || c.RealInventoryReadEnabled || c.RealProductDraftWriteEnabled || c.InventoryMutationEnabled || c.BackgroundWorkerEnabled || c.AutomaticRetryEnabled
	if level == "L0" && anyRealCapability {
		return fmt.Errorf("%s: all real, mutation, worker, and retry capabilities must remain disabled at L0", ErrCodeP10BoundaryViolation)
	}
	if level == "L1" && (c.RealProductDraftWriteEnabled || c.InventoryMutationEnabled || c.AutomaticRetryEnabled) {
		return fmt.Errorf("%s: write and retry capabilities are forbidden at L1", ErrCodeP10BoundaryViolation)
	}
	if level == "L3" && (!c.RealProviderEnabled || !c.RealPlatformNetworkEnabled || !c.RealCredentialsEnabled || !c.RealProductDraftWriteEnabled || !c.BackgroundWorkerEnabled) {
		return fmt.Errorf("%s: L3 product draft writes require provider, network, credentials, draft-write, and worker controls", ErrCodeP10BoundaryViolation)
	}
	if c.RealProductDraftWriteEnabled && c.AutomaticRetryEnabled {
		return fmt.Errorf("%s: automatic retry is forbidden for real product draft writes", ErrCodeP10BoundaryViolation)
	}
	if c.OfflineOAuthEnabled && IsStagingOrProduction(appEnv) {
		return fmt.Errorf("%s: offline OAuth is development/test only", ErrCodeP10BoundaryViolation)
	}
	if c.LocalCredentialKey != "" && IsStagingOrProduction(appEnv) {
		return fmt.Errorf("%s: local credential keys are forbidden in staging/production", ErrCodeP10BoundaryViolation)
	}
	oauthStateTTL := c.OAuthStateTTL
	if oauthStateTTL == 0 {
		oauthStateTTL = 10 * time.Minute
	}
	if oauthStateTTL < time.Minute || oauthStateTTL > 30*time.Minute {
		return fmt.Errorf("%s: OAuth state TTL must be between 60 and 1800 seconds", ErrCodeP10BoundaryViolation)
	}
	requestTimeout := c.ProviderRequestTimeout
	if requestTimeout == 0 {
		requestTimeout = 30 * time.Second
	}
	connectTimeout := c.ProviderConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = 5 * time.Second
	}
	responseHeaderTimeout := c.ProviderResponseHeaderTime
	if responseHeaderTimeout == 0 {
		responseHeaderTimeout = 15 * time.Second
	}
	if requestTimeout < time.Second || requestTimeout > 2*time.Minute || connectTimeout <= 0 || responseHeaderTimeout <= 0 {
		return fmt.Errorf("%s: provider timeout configuration is invalid", ErrCodeP10BoundaryViolation)
	}
	maxResponseBytes := c.ProviderMaxResponseBytes
	if maxResponseBytes == 0 {
		maxResponseBytes = 2 * 1024 * 1024
	}
	providerConcurrency := c.ProviderConcurrency
	if providerConcurrency == 0 {
		providerConcurrency = 2
	}
	pageSize := c.SKUPageSize
	if pageSize == 0 {
		pageSize = 50
	}
	paginationLimit := c.PaginationLimit
	if paginationLimit == 0 {
		paginationLimit = 100
	}
	if maxResponseBytes < 64*1024 || maxResponseBytes > 10*1024*1024 || providerConcurrency < 1 || providerConcurrency > 10 || pageSize < 1 || pageSize > 100 || paginationLimit < 1 || paginationLimit > 100 {
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
