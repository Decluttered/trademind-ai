package disasterrecovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/config"
)

func TestProductionRejectsDisasterRecoveryDrillRecording(t *testing.T) {
	svc := &Service{Cfg: &config.Config{AppEnv: config.EnvProduction}}

	_, err := svc.CreateDrill(context.Background(), DrillRequest{ConfirmedIsolated: true}, nil)
	require.ErrorContains(t, err, "unavailable in production")
}
