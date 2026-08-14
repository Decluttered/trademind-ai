package collect

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/gorm"
)

func newCollectTenantScopeDB(t *testing.T) *gorm.DB {
	t.Helper()
	safeName := strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:collect_tenant_%s_%s?mode=memory", safeName, uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&CollectBatch{}, &CollectTask{}, &CollectTaskEvent{}))
	return db
}

func TestReconcileCollectBatchIgnoresCrossTenantChildRows(t *testing.T) {
	db := newCollectTenantScopeDB(t)
	svc := &Service{DB: db}
	batchID := uuid.New()
	require.NoError(t, db.Create(&CollectBatch{
		HardDeleteBase: model.HardDeleteBase{ID: batchID}, TenantID: 11, Source: "1688",
		TotalCount: 1, PendingCount: 1, Status: BatchStatusRunning,
	}).Error)
	for _, task := range []CollectTask{
		{
			HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 11, BatchID: &batchID,
			Source: "1688", SourceURL: "https://detail.1688.com/offer/1.html", Status: StatusSuccess,
		},
		{
			HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 22, BatchID: &batchID,
			Source: "1688", SourceURL: "https://detail.1688.com/offer/2.html", Status: StatusFailed,
		},
	} {
		require.NoError(t, db.Create(&task).Error)
	}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return svc.reconcileCollectBatchTx(context.Background(), tx, batchID)
	}))
	var batch CollectBatch
	require.NoError(t, db.First(&batch, "id = ?", batchID).Error)
	require.Equal(t, BatchStatusSuccess, batch.Status)
	require.Equal(t, 1, batch.SuccessCount)
	require.Zero(t, batch.FailedCount)
}

func TestSameURLSuccessHintIsTenantScoped(t *testing.T) {
	db := newCollectTenantScopeDB(t)
	svc := &Service{DB: db}
	const sourceURL = "https://detail.1688.com/offer/1.html"
	failed := CollectTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 11,
		Source: "1688", SourceURL: sourceURL, Status: StatusFailed, ErrorMessage: "TIMEOUT",
	}
	otherTenantSuccess := CollectTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 22,
		Source: "1688", SourceURL: sourceURL, Status: StatusSuccess,
	}
	require.NoError(t, db.Create(&failed).Error)
	require.NoError(t, db.Create(&otherTenantSuccess).Error)

	dto := svc.enrichTaskDTO(context.Background(), &failed)
	require.False(t, dto.SameURLSucceededElsewhere)

	sameTenantSuccess := CollectTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 11,
		Source: "1688", SourceURL: sourceURL, Status: StatusSuccess,
	}
	require.NoError(t, db.Create(&sameTenantSuccess).Error)
	dto = svc.enrichTaskDTO(context.Background(), &failed)
	require.True(t, dto.SameURLSucceededElsewhere)
}

func TestCancelRemainingBatchTasksDoesNotCrossTenant(t *testing.T) {
	db := newCollectTenantScopeDB(t)
	svc := &Service{DB: db}
	batchID := uuid.New()
	require.NoError(t, db.Create(&CollectBatch{
		HardDeleteBase: model.HardDeleteBase{ID: batchID}, TenantID: 11, Source: "taobao_tmall",
		TotalCount: 1, PendingCount: 1, Status: BatchStatusRunning,
	}).Error)
	sameTenant := CollectTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 11, BatchID: &batchID,
		Source: "taobao_tmall", SourceURL: "https://item.taobao.com/item.htm?id=1", Status: StatusPending,
	}
	otherTenant := CollectTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 22, BatchID: &batchID,
		Source: "taobao_tmall", SourceURL: "https://item.taobao.com/item.htm?id=2", Status: StatusPending,
	}
	require.NoError(t, db.Create(&sameTenant).Error)
	require.NoError(t, db.Create(&otherTenant).Error)

	svc.cancelRemainingBatchTasks(context.Background(), 11, batchID, "batch paused")
	require.NoError(t, db.First(&sameTenant, "id = ?", sameTenant.ID).Error)
	require.NoError(t, db.First(&otherTenant, "id = ?", otherTenant.ID).Error)
	require.Equal(t, StatusCancelled, sameTenant.Status)
	require.Equal(t, StatusPending, otherTenant.Status)
}
