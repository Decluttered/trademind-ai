package shop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	platformebay "github.com/trademind-ai/trademind/backend/internal/providers/platform/ebay"
)

const ebayOAuthRedisPrefix = "oauth:ebay:state:"

type EbayAuthorizeURLResult struct {
	AuthorizeURL string `json:"authorizeUrl"`
	State        string `json:"state"`
}

type EbayOAuthCallbackBody struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

func (s *Service) ebayRuntime(ctx context.Context, shopID uuid.UUID) (platformebay.RuntimeConfig, error) {
	_, auth, err := s.PlainAuthForProviderCtx(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, err
	}
	if auth.Extra == nil {
		auth.Extra = map[string]string{}
	}
	if s.Settings != nil {
		plain, readErr := s.Settings.PlainByGroup(ctx, 0, "platform_ebay")
		if readErr != nil {
			return platformebay.RuntimeConfig{}, readErr
		}
		for key, value := range plain {
			auth.Extra[key] = value
		}
	}
	if strings.TrimSpace(auth.Extra["environment"]) == "" {
		auth.Extra["environment"] = strings.TrimSpace(s.EbayEnv)
	}
	return platformebay.ResolveRuntime(auth)
}

// EbayPublishCredentials returns a current user token and the resolved eBay
// runtime without exposing either value through an HTTP response. Refreshes
// are persisted through the encrypted shop_auth_tokens path.
func (s *Service) EbayPublishCredentials(ctx context.Context, shopID uuid.UUID) (platformebay.RuntimeConfig, string, error) {
	_, auth, err := s.PlainAuthForProviderCtx(ctx, shopID)
	if err != nil {
		return platformebay.RuntimeConfig{}, "", err
	}
	if auth.Extra == nil {
		auth.Extra = map[string]string{}
	}
	if s.Settings != nil {
		plain, readErr := s.Settings.PlainByGroup(ctx, 0, "platform_ebay")
		if readErr != nil {
			return platformebay.RuntimeConfig{}, "", readErr
		}
		for key, value := range plain {
			auth.Extra[key] = value
		}
	}
	if strings.TrimSpace(auth.Extra["environment"]) == "" {
		auth.Extra["environment"] = strings.TrimSpace(s.EbayEnv)
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
	refreshCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	bundle, err := platformebay.RefreshAccessToken(refreshCtx, cfg, auth.RefreshToken)
	if err != nil {
		_ = s.setAuthStatusCtx(ctx, shopID, AuthError)
		return platformebay.RuntimeConfig{}, "", &platformebay.APIError{Class: platformebay.ErrorAuth, Message: "AUTH_REQUIRED: " + err.Error()}
	}
	if err := s.persistOAuthTokenRefresh(ctx, shopID, bundle.AccessToken, bundle.RefreshToken, &bundle.ExpiresAt, auth.RefreshTokenExpiresAt); err != nil {
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
	scopes := make([]any, len(token.Scopes))
	for index, scope := range token.Scopes {
		scopes[index] = scope
	}
	_, err = s.UpdateAuth(callback, shopID, UpdateAuthBody{AuthType: "oauth2", AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: &token.ExpiresAt, MarketplaceID: cfg.Marketplace, Scopes: scopes, AuthConfig: map[string]any{"environment": cfg.Environment, "marketplace_id": cfg.Marketplace}}, adminID)
	if err != nil {
		return nil, err
	}
	_ = s.setAuthStatusCtx(ctx, shopID, AuthAuthorized)
	return s.GetDetail(callback, shopID)
}
