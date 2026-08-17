package ebay

import (
	"testing"

	"github.com/stretchr/testify/require"
	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

func TestResolveRuntimeRequiresMarketplace(t *testing.T) {
	_, err := ResolveRuntime(platformp.TestConnectionRequest{Extra: map[string]string{"redirect_uri": "App-App-SB-1"}})
	require.ErrorContains(t, err, "marketplace_id is required")
}

func TestResolveRuntimeRejectsHTTPSRuName(t *testing.T) {
	_, err := ResolveRuntime(platformp.TestConnectionRequest{
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"redirect_uri": "https://admin.example.com/callback"},
	})
	require.ErrorContains(t, err, "RuName")
}

func TestResolveRuntimeAcceptsRuName(t *testing.T) {
	cfg, err := ResolveRuntime(platformp.TestConnectionRequest{
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"redirect_uri": "MindBay-MindBay-SB-abcd", "environment": "sandbox"},
	})
	require.NoError(t, err)
	require.Equal(t, "sandbox", cfg.Environment)
	require.Equal(t, "EBAY_DE", cfg.Marketplace)
	require.Equal(t, "MindBay-MindBay-SB-abcd", cfg.RedirectURI)
	require.Equal(t, "https://api.sandbox.ebay.com", cfg.APIBaseURL)
}

func TestResolveRuntimeMapsLegacySandboxFlag(t *testing.T) {
	cfg, err := ResolveRuntime(platformp.TestConnectionRequest{
		MarketplaceID: "EBAY_DE",
		Extra:         map[string]string{"sandbox_enabled": "false"},
	})
	require.NoError(t, err)
	require.Equal(t, "production", cfg.Environment)
	require.Equal(t, "https://api.ebay.com", cfg.APIBaseURL)
}

func TestRuntimeFromMergedMapRequiresRuName(t *testing.T) {
	_, err := RuntimeFromMergedMap(map[string]string{
		"client_id": "id", "client_secret": "secret", "marketplace_id": "EBAY_DE", "environment": "sandbox",
	})
	require.ErrorContains(t, err, "RuName")
}

func TestRefreshScopesOmitsAccountReadonlyForLegacyTokens(t *testing.T) {
	require.NotContains(t, RefreshScopes(nil), AccountReadonlyScope)
	require.Contains(t, RefreshScopes([]string{AccountReadonlyScope, InventoryScope}), AccountReadonlyScope)
}

func TestHasAccountReadonlyScope(t *testing.T) {
	require.False(t, HasAccountReadonlyScope(nil))
	require.True(t, HasAccountReadonlyScope(AuthorizationScopes))
}

func TestMarketplaceRegionCurrency(t *testing.T) {
	region, currency := MarketplaceRegionCurrency("EBAY_DE", "")
	require.Equal(t, "DE", region)
	require.Equal(t, "EUR", currency)
	_, currency = MarketplaceRegionCurrency("EBAY_DE", "USD")
	require.Equal(t, "USD", currency)
}
