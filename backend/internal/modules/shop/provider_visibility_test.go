package shop

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/config"
	platformp "github.com/trademind-ai/trademind/backend/internal/providers/platform"
)

func TestProductionPlatformProvidersExcludeMockAndPlanned(t *testing.T) {
	platformp.Bootstrap()

	production := (&Service{AppEnv: config.EnvProduction}).ListPlatformProviders()
	require.NotEmpty(t, production)
	for _, provider := range production {
		require.NotEqual(t, "mock", provider.Platform)
		require.Contains(t, []string{platformp.StatusAvailable, platformp.StatusBeta}, provider.Status)
	}

	development := (&Service{AppEnv: config.EnvDevelopment}).ListPlatformProviders()
	require.True(t, hasPlatformProvider(development, "mock"))
	require.True(t, hasPlatformProvider(development, "shopify"))
}

func TestProductionProviderGuardRejectsMockAndPlanned(t *testing.T) {
	platformp.Bootstrap()
	svc := &Service{AppEnv: config.EnvProduction}

	require.Error(t, svc.requireProviderAvailable(platformp.MustGet("mock")))
	require.Error(t, svc.requireProviderAvailable(platformp.MustGet("shopify")))
	require.NoError(t, svc.requireProviderAvailable(platformp.MustGet("manual")))
}

func hasPlatformProvider(providers []PlatformProviderDTO, platform string) bool {
	for _, provider := range providers {
		if provider.Platform == platform {
			return true
		}
	}
	return false
}
