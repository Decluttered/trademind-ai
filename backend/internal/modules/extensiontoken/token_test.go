package extensiontoken

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"testing"
	"time"
)

func TestExtensionTokenHasDedicatedAudience(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret-that-is-long-enough-for-tests"}
	grant := Grant{WorkspaceID: 5, AdminUserID: uuid.New(), JTI: uuid.NewString(), Scope: "capture", ExpiresAt: time.Now().Add(time.Minute)}
	raw, err := Mint(cfg, grant)
	require.NoError(t, err)
	claims, err := Parse(cfg, raw)
	require.NoError(t, err)
	require.Equal(t, int64(5), claims.WorkspaceID)
	require.Equal(t, "extension_capture", claims.TokenType)
}
