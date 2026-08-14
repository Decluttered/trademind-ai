package database

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/productpublish"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	"gorm.io/gorm"
)

func TestMigrateProductPublishTenantBackfillsKnownOwnershipAndIsIdempotent(t *testing.T) {
	safeName := strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s_%s?mode=memory", safeName, uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(
		&product.Product{},
		&productpublish.ProductPublishTask{},
		&productpublish.ProductPublication{},
		&productpublish.ProductPublishBatch{},
	))

	const tenantID int64 = 42
	productID, otherProductID, shopID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Create(&product.Product{
		Base: model.Base{ID: productID}, TenantID: tenantID, Source: "manual", Title: "Tenant Product", Status: product.StatusDraft,
	}).Error)
	require.NoError(t, db.Create(&product.Product{
		Base: model.Base{ID: otherProductID}, TenantID: 84, Source: "manual", Title: "Other Tenant Product", Status: product.StatusDraft,
	}).Error)

	productBatchID, taskBatchID, mixedBatchID, conflictingBatchID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	productBatch := productpublish.ProductPublishBatch{
		HardDeleteBase: model.HardDeleteBase{ID: productBatchID}, ProductID: &productID, Status: productpublish.BatchFailed,
	}
	taskBatch := productpublish.ProductPublishBatch{
		HardDeleteBase: model.HardDeleteBase{ID: taskBatchID}, Status: productpublish.BatchFailed,
	}
	mixedBatch := productpublish.ProductPublishBatch{
		HardDeleteBase: model.HardDeleteBase{ID: mixedBatchID}, Status: productpublish.BatchFailed,
	}
	conflictingBatch := productpublish.ProductPublishBatch{
		HardDeleteBase: model.HardDeleteBase{ID: conflictingBatchID}, ProductID: &productID, Status: productpublish.BatchFailed,
	}
	require.NoError(t, db.Create(&productBatch).Error)
	require.NoError(t, db.Create(&taskBatch).Error)
	require.NoError(t, db.Create(&mixedBatch).Error)
	require.NoError(t, db.Create(&conflictingBatch).Error)

	taskID := uuid.New()
	require.NoError(t, db.Create(&productpublish.ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: taskID}, ProductID: productID, ShopID: shopID, TargetStoreID: shopID,
		BatchID: &taskBatchID, Platform: "shopee", TaskType: productpublish.TaskTypeLocalDraftCreate,
		Status: productpublish.TaskFailed, Mode: productpublish.PublishModeSaveAsPlatformDraft,
	}).Error)
	for _, task := range []productpublish.ProductPublishTask{
		{
			HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, ProductID: productID, ShopID: shopID, TargetStoreID: shopID,
			BatchID: &mixedBatchID, Platform: "shopee", TaskType: productpublish.TaskTypeLocalDraftCreate,
			Status: productpublish.TaskFailed, Mode: productpublish.PublishModeSaveAsPlatformDraft,
		},
		{
			HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, ProductID: otherProductID, ShopID: shopID, TargetStoreID: shopID,
			BatchID: &mixedBatchID, Platform: "shopee", TaskType: productpublish.TaskTypeLocalDraftCreate,
			Status: productpublish.TaskFailed, Mode: productpublish.PublishModeSaveAsPlatformDraft,
		},
		{
			HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, ProductID: otherProductID, ShopID: shopID, TargetStoreID: shopID,
			BatchID: &conflictingBatchID, Platform: "shopee", TaskType: productpublish.TaskTypeLocalDraftCreate,
			Status: productpublish.TaskFailed, Mode: productpublish.PublishModeSaveAsPlatformDraft,
		},
	} {
		require.NoError(t, db.Create(&task).Error)
	}
	publicationID := uuid.New()
	require.NoError(t, db.Create(&productpublish.ProductPublication{
		Base: model.Base{ID: publicationID}, ProductID: productID, ShopID: shopID, Platform: "shopee",
		Status: productpublish.StatusDraft, PublishStatus: productpublish.StatusDraftCreated,
	}).Error)

	orphanProductID := uuid.New()
	orphanTaskID, orphanPublicationID, orphanBatchID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Create(&productpublish.ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: orphanTaskID}, ProductID: orphanProductID, ShopID: shopID, TargetStoreID: shopID,
		Platform: "shopee", TaskType: productpublish.TaskTypeLocalDraftCreate, Status: productpublish.TaskFailed,
		Mode: productpublish.PublishModeSaveAsPlatformDraft,
	}).Error)
	require.NoError(t, db.Create(&productpublish.ProductPublication{
		Base: model.Base{ID: orphanPublicationID}, ProductID: orphanProductID, ShopID: shopID, Platform: "shopee",
		Status: productpublish.StatusDraft, PublishStatus: productpublish.StatusDraftCreated,
	}).Error)
	require.NoError(t, db.Create(&productpublish.ProductPublishBatch{
		HardDeleteBase: model.HardDeleteBase{ID: orphanBatchID}, ProductID: &orphanProductID, Status: productpublish.BatchFailed,
	}).Error)

	require.NoError(t, migrateProductPublishTenant(db))
	require.NoError(t, migrateProductPublishTenant(db))

	assertTenant := func(dst any, id uuid.UUID, want int64) {
		t.Helper()
		require.NoError(t, db.First(dst, "id = ?", id).Error)
		switch row := dst.(type) {
		case *productpublish.ProductPublishTask:
			require.Equal(t, want, row.TenantID)
		case *productpublish.ProductPublication:
			require.Equal(t, want, row.TenantID)
		case *productpublish.ProductPublishBatch:
			require.Equal(t, want, row.TenantID)
		default:
			t.Fatalf("unsupported tenant model %T", dst)
		}
	}
	assertTenant(&productpublish.ProductPublishTask{}, taskID, tenantID)
	assertTenant(&productpublish.ProductPublication{}, publicationID, tenantID)
	assertTenant(&productpublish.ProductPublishBatch{}, productBatchID, tenantID)
	assertTenant(&productpublish.ProductPublishBatch{}, taskBatchID, tenantID)
	assertTenant(&productpublish.ProductPublishBatch{}, mixedBatchID, 0)
	assertTenant(&productpublish.ProductPublishBatch{}, conflictingBatchID, 0)
	assertTenant(&productpublish.ProductPublishTask{}, orphanTaskID, 0)
	assertTenant(&productpublish.ProductPublication{}, orphanPublicationID, 0)
	assertTenant(&productpublish.ProductPublishBatch{}, orphanBatchID, 0)
}
