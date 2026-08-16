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

func BuildAuthorizeURL(cfg RuntimeConfig, state string) (string, error) {
	if cfg.ClientID == "" || cfg.RedirectURI == "" || strings.TrimSpace(state) == "" {
		return "", fmt.Errorf("eBay client_id, redirect_uri and OAuth state are required")
	}
	values := url.Values{"client_id": {cfg.ClientID}, "redirect_uri": {cfg.RedirectURI}, "response_type": {"code"}, "scope": {strings.Join(AuthorizationScopes, " ")}, "state": {state}}
	return cfg.AuthBaseURL + "/oauth2/authorize?" + values.Encode(), nil
}

func ExchangeAuthCode(ctx context.Context, cfg RuntimeConfig, code string) (TokenBundle, error) {
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"authorization_code"}, "code": {strings.TrimSpace(code)}, "redirect_uri": {cfg.RedirectURI}})
}

func RefreshAccessToken(ctx context.Context, cfg RuntimeConfig, refreshToken string) (TokenBundle, error) {
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {strings.TrimSpace(refreshToken)}, "scope": {strings.Join(AuthorizationScopes, " ")}})
}

func ApplicationToken(ctx context.Context, cfg RuntimeConfig) (TokenBundle, error) {
	return exchangeToken(ctx, cfg, url.Values{"grant_type": {"client_credentials"}, "scope": {"https://api.ebay.com/oauth/api_scope"}})
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
