package productpublish

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
)

func TestTraditionalPublishAllowedByEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		allowed     bool
	}{
		{name: "empty development default", environment: "", allowed: true},
		{name: "development", environment: "development", allowed: true},
		{name: "test", environment: "test", allowed: true},
		{name: "staging", environment: " staging ", allowed: false},
		{name: "production", environment: "PRODUCTION", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{Environment: tt.environment}
			if got := svc.traditionalPublishAllowed(); got != tt.allowed {
				t.Fatalf("traditionalPublishAllowed() = %v, want %v", got, tt.allowed)
			}
		})
	}

	var nilService *Service
	if nilService.traditionalPublishAllowed() {
		t.Fatal("nil service must fail closed")
	}
}

func TestRetryFailedTraditionalTaskIsRejectedWithoutMutationInProduction(t *testing.T) {
	db := newBatchIntegrationDB(t)
	svc := newBatchTestService(db)
	svc.Environment = "production"
	shopID := uuid.New()
	task := ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()},
		TenantID:       0,
		ProductID:      uuid.New(),
		ShopID:         shopID,
		TargetStoreID:  shopID,
		Platform:       "shopee",
		TaskType:       TaskTypeProductPublish,
		Status:         TaskFailed,
		PublishStatus:  StatusPubFailed,
		Retryable:      true,
		ErrorCode:      "LEGACY_FAILURE",
		ErrorMessage:   "legacy failure",
	}
	require.NoError(t, db.Create(&task).Error)

	_, err := svc.RetryFailed(testGinContext(), task.ID, nil)
	require.ErrorIs(t, err, ErrTraditionalPublishProductionDisabled)

	var stored ProductPublishTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.Equal(t, TaskFailed, stored.Status)
	require.Equal(t, StatusPubFailed, stored.PublishStatus)
	require.Equal(t, "LEGACY_FAILURE", stored.ErrorCode)
	require.Equal(t, "legacy failure", stored.ErrorMessage)
	require.True(t, stored.Retryable)
}

func TestGenericWorkerFailsTraditionalTaskBeforeProductOrProviderAccessInProduction(t *testing.T) {
	db := newBatchIntegrationDB(t)
	svc := newBatchTestService(db)
	svc.Environment = "staging"
	shopID := uuid.New()
	task := ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()},
		TenantID:       0,
		ProductID:      uuid.New(),
		ShopID:         shopID,
		TargetStoreID:  shopID,
		Platform:       "shopee",
		TaskType:       TaskTypeProductPublish,
		Status:         TaskPending,
		PublishStatus:  StatusReady,
		Retryable:      true,
	}
	require.NoError(t, db.Create(&task).Error)

	err := svc.ProcessQueuedTask(context.Background(), task.ID, "production-guard-test-worker")
	require.ErrorIs(t, err, ErrTraditionalPublishProductionDisabled)

	var stored ProductPublishTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.Equal(t, TaskFailed, stored.Status)
	require.Equal(t, StatusPubFailed, stored.PublishStatus)
	require.Equal(t, ErrorTraditionalPublishProductionDisabled, stored.ErrorCode)
	require.Equal(t, ErrTraditionalPublishProductionDisabled.Error(), stored.ErrorMessage)
	require.NotNil(t, stored.FinishedAt)
}
