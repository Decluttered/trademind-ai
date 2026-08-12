package tasktenant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/pkg/security"
)

func TestBeginWorkerWithLegacyZeroRequiresExplicitAuthorization(t *testing.T) {
	_, _, err := BeginWorker(context.Background(), nil, 0, uuid.Nil, "collect")
	require.True(t, errors.Is(err, security.ErrTaskTenantMissing))

	ctx, scope, err := BeginWorkerWithLegacyZero(context.Background(), nil, 0, uuid.Nil, "collect", true)
	require.NoError(t, err)
	require.Zero(t, scope.TenantID)
	require.True(t, scope.AllowLegacyTenantZero)

	tenant, err := security.RequireTenantContext(ctx)
	require.NoError(t, err)
	require.Zero(t, tenant.TenantID)
	require.Equal(t, security.AuthSourceLegacyDevZero, tenant.AuthSource)
}

func TestBeginWorkerWithLegacyZeroStillRejectsNegativeTenant(t *testing.T) {
	_, _, err := BeginWorkerWithLegacyZero(context.Background(), nil, -1, uuid.Nil, "collect", true)
	require.True(t, errors.Is(err, security.ErrTaskTenantMissing))
}
