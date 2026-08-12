package collect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcceptsTenantIDRequiresExplicitLegacyZeroMode(t *testing.T) {
	require.True(t, (&Service{}).acceptsTenantID(42))
	require.False(t, (&Service{}).acceptsTenantID(0))
	require.True(t, (&Service{AllowLegacyTenantZero: true}).acceptsTenantID(0))
	require.False(t, (&Service{AllowLegacyTenantZero: true}).acceptsTenantID(-1))
}
