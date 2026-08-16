package ebay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

func TestConnectionWithoutUserTokenUsesApplicationGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/identity/v1/oauth2/token", request.URL.Path)
		raw, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.Contains(t, string(raw), "grant_type=client_credentials")
		require.NotContains(t, string(raw), "sell.account")
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"access_token": "app-token", "expires_in": 7200, "token_type": "Application Access Token"})
	}))
	t.Cleanup(server.Close)
	res, err := Provider{}.TestConnection(context.Background(), platformp.TestConnectionRequest{
		AppKey:        "client",
		AppSecret:     "secret",
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"api_base_url": server.URL, "environment": "sandbox", "timeout_sec": "5"},
	})
	require.NoError(t, err)
	require.True(t, res.OK)
	require.Contains(t, res.Message, "application token")
	require.Equal(t, "DE", res.Region)
	require.Equal(t, "EUR", res.Currency)
}

func TestConnectionWithUserTokenCallsGetPrivileges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/sell/account/v1/privilege", request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"sellerRegistrationCompleted":true,"sellingLimit":{"amount":{"currency":"EUR","value":"20.00"}}}`))
	}))
	t.Cleanup(server.Close)
	res, err := Provider{}.TestConnection(context.Background(), platformp.TestConnectionRequest{
		AccessToken:   "user-token",
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"api_base_url": server.URL, "environment": "sandbox", "timeout_sec": "5"},
	})
	require.NoError(t, err)
	require.True(t, res.OK)
	require.Equal(t, "EUR", res.Currency)
}

func TestConnectionAuthFailureAsksForReauthorize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(`{"errors":[{"message":"Access denied"}]}`))
	}))
	t.Cleanup(server.Close)
	res, err := Provider{}.TestConnection(context.Background(), platformp.TestConnectionRequest{
		AccessToken:   "user-token",
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"api_base_url": server.URL, "timeout_sec": "5"},
	})
	require.NoError(t, err)
	require.False(t, res.OK)
	require.Contains(t, res.Message, "EBAY_REAUTHORIZATION_REQUIRED")
}

func TestEbayAppSchemaUsesRuNameAndRequiredMarketplace(t *testing.T) {
	schema := platformp.EbayAppConfigSchema()
	fields := map[string]platformp.AppConfigField{}
	for _, field := range schema.Fields {
		fields[field.Name] = field
	}
	require.Contains(t, fields["redirect_uri"].Label, "RuName")
	require.True(t, fields["marketplace_id"].Required)
	require.Equal(t, "sandbox", fields["environment"].DefaultValue)
	_, ok := fields["sandbox_enabled"]
	require.False(t, ok)
}
