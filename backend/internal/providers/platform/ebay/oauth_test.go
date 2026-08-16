package ebay

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAuthorizeURLIncludesPreparedSellScopes(t *testing.T) {
	raw, err := BuildAuthorizeURL(RuntimeConfig{ClientID: "client", RedirectURI: "ru-name", AuthBaseURL: "https://auth.sandbox.ebay.test"}, "state-value")
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "state-value", parsed.Query().Get("state"))
	scopes := strings.Fields(parsed.Query().Get("scope"))
	for _, expected := range AuthorizationScopes {
		require.Contains(t, scopes, expected)
	}
	require.Contains(t, scopes, AccountReadonlyScope)
}

func TestBuildAuthorizeURLRejectsHTTPSRuName(t *testing.T) {
	_, err := BuildAuthorizeURL(RuntimeConfig{ClientID: "client", RedirectURI: "https://example.com/cb", AuthBaseURL: "https://auth.sandbox.ebay.test"}, "state")
	require.ErrorContains(t, err, "RuName")
}
