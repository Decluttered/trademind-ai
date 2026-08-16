package shop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
	platformebay "github.com/trademind-ai/trademind/backend/internal/providers/platform/ebay"
	"gorm.io/datatypes"
)

const (
	ebayOAuthRedisPrefix   = "oauth:ebay:state:"
	ebayRefreshLockPrefix  = "ebay:token-refresh:"
	ebayRefreshLockTTL     = 30 * time.Second
	ebayRefreshLockRetries = 4
	ebayRefreshLockWait    = 50 * time.Millisecond
)

type EbayAuthorizeURLResult struct {
	AuthorizeURL string `json:"authorizeUrl"`
	State        string `json:"state"`
}

type EbayOAuthCallbackBody struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

func (s *Service) mergeEbayAuth(ctx context.Context, auth platformp.TestConnectionRequest) (platformp.TestConnectionRequest, error) {
	if auth.Extra == nil {
		auth.Extra = map[string]string{}
	}
	if s.Settings != nil {
		plain, readErr := s.Settings.PlainByGroup(ctx, 0, "platform_ebay")
		if readErr != nil {
			return auth, readErr
		}
		for key, value := range plain {
			if strings.TrimSpace(auth.Extra[key]) == "" && strings.TrimSpace(value) != "" {
				auth.Extra[key] = value
			}
		}
	}
	if strings.TrimSpace(auth.Extra["environment"]) == "" {
		auth.Extra["environment"] = strings.TrimSpace(s.EbayEnv)
	}
	return auth, nil
}

func (s *Service) ebayRuntime(ctx context.Context, shopID uuid.UUID) (platformebay.RuntimeConfig, error) {
	_, auth, err := s.PlainAuthForProviderCtx(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, err
	}
	auth, err = s.mergeEbayAuth(ctx, auth)
	if err != nil {
		return platformebay.RuntimeConfig{}, err
	}
	return platformebay.ResolveRuntime(auth)
}

// EbayPublishCredentials returns a current user token and the resolved eBay
// runtime without exposing either value through an HTTP response. Refreshes
// are persisted through the encrypted shop_auth_tokens path and use a per-shop Redis lock.
func (s *Service) EbayPublishCredentials(ctx context.Context, shopID uuid.UUID) (platformebay.RuntimeConfig, string, error) {
	_, auth, err := s.PlainAuthForProviderCtx(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	auth, err = s.mergeEbayAuth(ctx, auth)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	cfg, err := platformebay.ResolveRuntime(auth)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	if strings.TrimSpace(auth.AccessToken) != "" && (auth.AccessTokenExpiresAt == nil || auth.AccessTokenExpiresAt.After(time.Now().UTC().Add(2*time.Minute))) {
		return cfg, auth.AccessToken, nil
	}
	if strings.TrimSpace(auth.RefreshToken) == "" {
		return platformebay.RuntimeConfig{}, "", &platformebay.APIError{Class: platformebay.ErrorAuth, Message: "AUTH_REQUIRED: eBay refresh token is missing"}
	}
	unlock, err := s.acquireEbayRefreshLock(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	if unlock != nil {
		defer unlock()
	}
	_, auth, err = s.PlainAuthForProviderCtx(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	auth, err = s.mergeEbayAuth(ctx, auth)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	cfg, err = platformebay.ResolveRuntime(auth)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	if strings.TrimSpace(auth.AccessToken) != "" && (auth.AccessTokenExpiresAt == nil || auth.AccessTokenExpiresAt.After(time.Now().UTC().Add(2*time.Minute))) {
		return cfg, auth.AccessToken, nil
	}
	granted := s.ebayStoredScopes(ctx, shopID)
	refreshCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	bundle, err := platformebay.RefreshAccessToken(refreshCtx, cfg, auth.RefreshToken, granted)
	if err != nil {
		_ = s.setAuthStatusCtx(ctx, shopID, AuthError)
		_ = s.setEbayReauthorization(ctx, shopID, true, "EBAY_AUTH_EXPIRED")
		return platformebay.RuntimeConfig{}, "", &platformebay.APIError{Class: platformebay.ErrorAuth, Message: "AUTH_REQUIRED: " + err.Error()}
	}
	if err := s.persistOAuthTokenRefresh(ctx, shopID, bundle.AccessToken, bundle.RefreshToken, &bundle.ExpiresAt, auth.RefreshTokenExpiresAt); err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	if err := s.persistEbayGrantedScopes(ctx, shopID, bundle.Scopes); err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	return cfg, bundle.AccessToken, nil
}

func (s *Service) EbayOAuthAuthorizeURL(c *gin.Context, shopID uuid.UUID) (*EbayAuthorizeURLResult, error) {
	if s.Redis == nil || s.Redis.Client == nil {
		return nil, fmt.Errorf("redis required for OAuth state")
	}
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		return nil, err
	}
	if !adminperm.RequireStoreOperate(c, s.DB, shopID) {
		return nil, fmt.Errorf("shop not found")
	}
	var row Shop
	if err := s.DB.WithContext(c).First(&row, "id=? AND tenant_id=?", shopID, tenantID).Error; err != nil {
		return nil, err
	}
	if row.Platform != "ebay" {
		return nil, fmt.Errorf("shop platform must be ebay")
	}
	state, err := randomOAuthState()
	if err != nil {
		return nil, err
	}
	cfg, err := s.ebayRuntime(c, shopID)
	if err != nil {
		return nil, err
	}
	authorizeURL, err := platformebay.BuildAuthorizeURL(cfg, state)
	if err != nil {
		return nil, err
	}
	if err := s.Redis.Set(c, ebayOAuthRedisPrefix+state, shopID.String(), 10*time.Minute).Err(); err != nil {
		return nil, err
	}
	return &EbayAuthorizeURLResult{AuthorizeURL: authorizeURL, State: state}, nil
}

func (s *Service) EbayOAuthCallback(c *gin.Context, shopID uuid.UUID, body EbayOAuthCallbackBody, adminID *uuid.UUID) (*ShopDetailDTO, error) {
	tenantID, err := adminperm.TenantIDFromGin(c)
	if err != nil {
		return nil, err
	}
	if !adminperm.RequireStoreOperate(c, s.DB, shopID) {
		return nil, fmt.Errorf("shop not found")
	}
	var shopRow Shop
	if err := s.DB.WithContext(c).First(&shopRow, "id=? AND tenant_id=? AND platform=?", shopID, tenantID, "ebay").Error; err != nil {
		return nil, err
	}
	code, state := strings.TrimSpace(body.Code), strings.TrimSpace(body.State)
	if code == "" || state == "" {
		return nil, fmt.Errorf("code and state required")
	}
	if s.Redis == nil || s.Redis.Client == nil {
		return nil, fmt.Errorf("redis required for OAuth state")
	}
	key := ebayOAuthRedisPrefix + state
	saved, err := s.Redis.Get(c, key).Result()
	if err != nil || saved != shopID.String() {
		return nil, fmt.Errorf("invalid or expired oauth state")
	}
	_ = s.Redis.Del(c, key).Err()
	cfg, err := s.ebayRuntime(c, shopID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(c, cfg.Timeout)
	defer cancel()
	token, err := platformebay.ExchangeAuthCode(ctx, cfg, code)
	if err != nil {
		_ = s.setAuthStatusCtx(ctx, shopID, AuthError)
		return nil, err
	}
	callback := c.Copy()
	callback.Request = c.Request.WithContext(ctx)
	scopes := token.Scopes
	if len(scopes) == 0 {
		scopes = append([]string{}, platformebay.AuthorizationScopes...)
	}
	stored := make([]any, len(scopes))
	for index, scope := range scopes {
		stored[index] = scope
	}
	_, err = s.UpdateAuth(callback, shopID, UpdateAuthBody{AuthType: "oauth2", AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: &token.ExpiresAt, MarketplaceID: cfg.Marketplace, Scopes: stored, AuthConfig: map[string]any{"environment": cfg.Environment, "marketplace_id": cfg.Marketplace}}, adminID)
	if err != nil {
		return nil, err
	}
	_ = s.setAuthStatusCtx(ctx, shopID, AuthAuthorized)
	_ = s.setEbayReauthorization(ctx, shopID, false, "")
	return s.GetDetail(callback, shopID)
}

func (s *Service) acquireEbayRefreshLock(ctx context.Context, shopID uuid.UUID) (func(), error) {
	if s == nil || s.Redis == nil || s.Redis.Client == nil {
		return nil, nil
	}
	key := ebayRefreshLockPrefix + shopID.String()
	for attempt := 0; attempt < ebayRefreshLockRetries; attempt++ {
		ok, err := s.Redis.SetNX(ctx, key, "1", ebayRefreshLockTTL).Result()
		if err != nil {
			return nil, fmt.Errorf("eBay refresh lock: %w", err)
		}
		if ok {
			return func() {
				_ = s.Redis.Del(context.Background(), key).Err()
			}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(ebayRefreshLockWait):
		}
	}
	return nil, fmt.Errorf("eBay token refresh is already in progress; retry")
}

func (s *Service) ebayStoredScopes(ctx context.Context, shopID uuid.UUID) []string {
	var tok ShopAuthToken
	if err := s.DB.WithContext(ctx).Select("scopes").Where("shop_id = ?", shopID).First(&tok).Error; err != nil {
		return nil
	}
	return parseJSONStringSlice(tok.Scopes)
}

func parseJSONStringSlice(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func (s *Service) persistEbayGrantedScopes(ctx context.Context, shopID uuid.UUID, scopes []string) error {
	if len(scopes) == 0 {
		return nil
	}
	encoded, err := json.Marshal(scopes)
	if err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Model(&ShopAuthToken{}).Where("shop_id = ?", shopID).Update("scopes", datatypes.JSON(encoded)).Error
}

func (s *Service) setEbayReauthorization(ctx context.Context, shopID uuid.UUID, required bool, errCode string) error {
	updates := map[string]any{"reauthorization_required": required, "last_refresh_error_code": strings.TrimSpace(errCode)}
	return s.DB.WithContext(ctx).Model(&ShopAuthToken{}).Where("shop_id = ?", shopID).Updates(updates).Error
}

func (s *Service) persistEbayPrivilegeSnapshot(ctx context.Context, shopID uuid.UUID, cfg platformebay.RuntimeConfig, snapshot platformebay.PrivilegeSnapshot, region, currency string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("shop: no db")
	}
	if strings.TrimSpace(region) != "" || strings.TrimSpace(currency) != "" {
		shopUpdates := map[string]any{}
		if strings.TrimSpace(region) != "" {
			shopUpdates["region"] = region
		}
		if strings.TrimSpace(currency) != "" {
			shopUpdates["currency"] = currency
		}
		if err := s.DB.WithContext(ctx).Model(&Shop{}).Where("id = ?", shopID).Updates(shopUpdates).Error; err != nil {
			return err
		}
	}
	var tok ShopAuthToken
	if err := s.DB.WithContext(ctx).Where("shop_id = ?", shopID).First(&tok).Error; err != nil {
		return err
	}
	config := map[string]any{}
	if len(tok.AuthConfig) > 0 {
		_ = json.Unmarshal(tok.AuthConfig, &config)
	}
	if config == nil {
		config = map[string]any{}
	}
	config["environment"] = cfg.Environment
	config["marketplace_id"] = cfg.Marketplace
	if snapshot.SellerRegistrationCompleted != nil {
		config["sellerRegistrationCompleted"] = *snapshot.SellerRegistrationCompleted
	}
	if snapshot.SellingLimitAmount != "" {
		config["sellingLimitAmount"] = snapshot.SellingLimitAmount
	}
	if snapshot.SellingLimitCurrency != "" {
		config["sellingLimitCurrency"] = snapshot.SellingLimitCurrency
	}
	if snapshot.SellingLimitQuantity != nil {
		config["sellingLimitQuantity"] = *snapshot.SellingLimitQuantity
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Model(&tok).Update("auth_config", datatypes.JSON(encoded)).Error
}

func (s *Service) testEbayConnection(ctx context.Context, shopID uuid.UUID) (*platformp.TestConnectionResult, error) {
	cfg, access, err := s.EbayPublishCredentials(ctx, shopID)
	if err != nil {
		return &platformp.TestConnectionResult{OK: false, Message: err.Error()}, nil
	}
	if !platformebay.HasAccountReadonlyScope(s.ebayStoredScopes(ctx, shopID)) {
		_ = s.setEbayReauthorization(ctx, shopID, true, "EBAY_REAUTHORIZATION_REQUIRED")
		return &platformp.TestConnectionResult{OK: false, Message: platformebay.MsgReauthorizeAccountReadonly}, nil
	}
	res, snapshot, err := platformebay.ProbeUserConnection(ctx, cfg, access)
	if err != nil {
		return nil, err
	}
	if res != nil && !res.OK && strings.Contains(res.Message, "EBAY_REAUTHORIZATION_REQUIRED") {
		_ = s.setEbayReauthorization(ctx, shopID, true, "EBAY_REAUTHORIZATION_REQUIRED")
	}
	if res != nil && res.OK {
		_ = s.setEbayReauthorization(ctx, shopID, false, "")
		_ = s.persistEbayPrivilegeSnapshot(ctx, shopID, cfg, snapshot, res.Region, res.Currency)
	}
	return res, nil
}
