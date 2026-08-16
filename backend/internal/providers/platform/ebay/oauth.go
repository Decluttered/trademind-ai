package ebay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TokenBundle struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scopes       []string
}

// BuildAuthorizeURL builds the Authorization Code Grant consent URL.
// Official: OAuth authorization code grant
// GET {auth.ebay.com|auth.sandbox.ebay.com}/oauth2/authorize
// Query: client_id, redirect_uri (RuName), response_type=code, scope, state
func BuildAuthorizeURL(cfg RuntimeConfig, state string) (string, error) {
	if cfg.ClientID == "" || cfg.RedirectURI == "" || strings.TrimSpace(state) == "" {
		return "", fmt.Errorf("eBay client_id, redirect_uri and OAuth state are required")
	}
	if err := ValidateRuName(cfg.RedirectURI); err != nil {
		return "", err
	}
	values := url.Values{"client_id": {cfg.ClientID}, "redirect_uri": {cfg.RedirectURI}, "response_type": {"code"}, "scope": {strings.Join(AuthorizationScopes, " ")}, "state": {state}}
	return cfg.AuthBaseURL + "/oauth2/authorize?" + values.Encode(), nil
}

// ExchangeAuthCode exchanges an authorization code for user tokens.
// Official: OAuth authorization code grant — redeem the code
// POST {api.ebay.com|api.sandbox.ebay.com}/identity/v1/oauth2/token
// grant_type=authorization_code; redirect_uri is the RuName
func ExchangeAuthCode(ctx context.Context, cfg RuntimeConfig, code string) (TokenBundle, error) {
	if err := ValidateRuName(cfg.RedirectURI); err != nil {
		return TokenBundle{}, err
	}
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"authorization_code"}, "code": {strings.TrimSpace(code)}, "redirect_uri": {cfg.RedirectURI}})
}

// RefreshAccessToken refreshes a user access token.
// Official: OAuth authorization code grant — refresh token
// POST /identity/v1/oauth2/token  grant_type=refresh_token
// scope must be a subset of scopes granted at consent; extra scopes fail the refresh.
func RefreshAccessToken(ctx context.Context, cfg RuntimeConfig, refreshToken string, grantedScopes []string) (TokenBundle, error) {
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {strings.TrimSpace(refreshToken)}, "scope": {strings.Join(RefreshScopes(grantedScopes), " ")}})
}

// ApplicationToken obtains an application (client credentials) token.
// Official: OAuth client credentials grant
// POST /identity/v1/oauth2/token  grant_type=client_credentials
// scope: https://api.ebay.com/oauth/api_scope
func ApplicationToken(ctx context.Context, cfg RuntimeConfig) (TokenBundle, error) {
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"client_credentials"}, "scope": {applicationTokenScope}})
}

func exchangeToken(ctx context.Context, cfg RuntimeConfig, form url.Values) (TokenBundle, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return TokenBundle{}, fmt.Errorf("eBay client credentials are incomplete")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIBaseURL+"/identity/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return TokenBundle{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret)))
	client := &http.Client{Timeout: cfg.Timeout}
	res, err := client.Do(req)
	if err != nil {
		return TokenBundle{}, err
	}
	defer res.Body.Close()
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return TokenBundle{}, fmt.Errorf("decode eBay OAuth response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 || body.AccessToken == "" {
		return TokenBundle{}, fmt.Errorf("eBay OAuth failed (%d): %s %s", res.StatusCode, body.Error, body.Description)
	}
	return TokenBundle{AccessToken: body.AccessToken, RefreshToken: body.RefreshToken, ExpiresAt: time.Now().UTC().Add(time.Duration(body.ExpiresIn) * time.Second), Scopes: strings.Fields(body.Scope)}, nil
}
