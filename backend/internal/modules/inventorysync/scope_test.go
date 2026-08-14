package inventorysync

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInventorySyncModelsDoNotStoreCredentialFields(t *testing.T) {
	modelSource, err := os.ReadFile("model.go")
	require.NoError(t, err)
	source := strings.ToLower(string(modelSource))
	for _, forbidden := range []string{"accesstoken", "refresh_token", "refreshToken", "appsecret", "cookie", "oauth"} {
		require.NotContains(t, source, strings.ToLower(forbidden))
	}
}
