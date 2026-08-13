package productpublish

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationtask"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/productioncontrolp10"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/model"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func newDouyinPublishTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&product.Product{},
		&product.ProductSKU{},
		&product.ProductImage{},
		&product.ProductPlatformPublishConfig{},
		&ProductPublishTask{},
		&ProductPublication{},
		&ProductPublicationSKU{},
		&shop.Shop{},
		&shop.PlatformCategory{},
		&shop.PlatformCategoryAttribute{},
		&operationtask.OperationTask{},
		&operationtask.PlatformDraft{},
		&operationtask.ApprovalRecord{},
		&operationtask.ExecutionAttempt{},
		&operationtask.ExecutionOutbox{},
		&operationtask.ExecutionError{},
		&operationtask.OperationTaskEvent{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedBoundProductionDouyinTask(t *testing.T, db *gorm.DB) (*Service, ProductPublishTask, operationtask.FrozenDouyinDraft) {
	t.Helper()
	tenantID := int64(101)
	productID, shopID, operationTaskID, draftID, approvalID, attemptID, publicationID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	stock := 5
	mapping := product.DouyinDraftMapping{
		Platform: "douyin_shop", ProductID: productID.String(), ShopID: shopID.String(), CategoryID: "20219", Title: "Frozen title",
		SKUs: []product.DouyinDraftSKU{{LocalSkuID: uuid.NewString(), Name: "Default", Price: 99, Stock: &stock}},
	}
	request := platformdouyin.CreateProductDraftRequest{
		OuterProductID: productID.String(), Name: "Frozen title", CategoryLeafID: "20219",
		SpecPricesV2: []map[string]any{{"outer_sku_id": mapping.SKUs[0].LocalSkuID, "price": int64(9900), "stock_num": stock}},
	}
	mappingRaw, err := json.Marshal(mapping)
	require.NoError(t, err)
	requestRaw, err := json.Marshal(request)
	require.NoError(t, err)
	mappingHash, err := operationtask.ComputePayloadHash(mappingRaw)
	require.NoError(t, err)
	frozen := operationtask.FrozenDouyinDraft{
		SchemaVersion: operationtask.DouyinDraftSchemaVersion, ProductID: productID, ShopID: shopID,
		PublishMode: operationtask.DouyinDraftPublishMode, SKUCount: 1, Request: requestRaw,
		Review: json.RawMessage(`{"name":"Frozen title"}`), MappingSnapshot: mappingRaw, MappingHash: mappingHash,
	}
	frozenRaw, err := json.Marshal(frozen)
	require.NoError(t, err)
	payloadHash, err := operationtask.ComputePayloadHash(frozenRaw)
	require.NoError(t, err)
	require.NoError(t, db.Create(&operationtask.OperationTask{
		HardDeleteBase: model.HardDeleteBase{ID: operationTaskID}, TenantID: tenantID, SourceType: operationtask.OperationTaskSourceManual,
		SourceReference: productID.String(), TaskType: operationtask.OperationTaskTypeProductPublish, Platform: operationtask.PlatformDouyin,
		Title: "Production draft", Payload: datatypes.JSON([]byte(`{}`)), Status: operationtask.OperationTaskStatusExecutionQueued,
		Priority: operationtask.OperationTaskPriorityNormal, Revision: 2,
	}).Error)
	require.NoError(t, db.Create(&operationtask.PlatformDraft{
		HardDeleteBase: model.HardDeleteBase{ID: draftID}, TenantID: tenantID, OperationTaskID: operationTaskID, Platform: operationtask.PlatformDouyin,
		AdapterMode: operationtask.AdapterModeProductionDraft, DraftVersion: 1, Payload: datatypes.JSON(frozenRaw), PayloadHash: payloadHash,
		Status: operationtask.PlatformDraftStatusApproved,
	}).Error)
	require.NoError(t, db.Create(&operationtask.ApprovalRecord{
		HardDeleteBase: model.HardDeleteBase{ID: approvalID}, TenantID: tenantID, OperationTaskID: operationTaskID, PlatformDraftID: draftID,
		Decision: operationtask.ApprovalDecisionApproved, DraftVersion: 1, DraftPayloadHash: payloadHash,
		ReviewerID: uuid.New(), ReviewerRole: operationtask.ReviewerRoleAdmin,
	}).Error)
	require.NoError(t, db.Create(&operationtask.ExecutionAttempt{
		HardDeleteBase: model.HardDeleteBase{ID: attemptID}, TenantID: tenantID, OperationTaskID: operationTaskID, PlatformDraftID: draftID,
		ApprovalRecordID: approvalID, AttemptNumber: 1, Status: operationtask.ExecutionAttemptStatusQueued,
		AdapterMode: operationtask.AdapterModeProductionDraft, Platform: operationtask.PlatformDouyin,
		ApprovedDraftVersion: 1, ApprovedDraftPayloadHash: payloadHash, ExecutedDraftVersion: 1, ExecutedDraftPayloadHash: payloadHash,
		Revision: 1,
	}).Error)
	require.NoError(t, db.Create(&ProductPublication{
		Base: model.Base{ID: publicationID}, ProductID: productID, ShopID: shopID, Platform: "douyin_shop",
		Status: StatusDraft, PublishStatus: StatusChecking,
	}).Error)
	snapshotRaw, err := json.Marshal(douyinDraftSnapshot{
		PublicationID: publicationID, PublishMode: PublishModeSaveAsPlatformDraft, MappingHash: mappingHash,
		FrozenRequest: &request, FrozenMapping: &mapping, ExecutionAttemptID: &attemptID, OperationTaskID: &operationTaskID,
		FrozenPayloadHash: payloadHash,
	})
	require.NoError(t, err)
	task := ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: tenantID, ProductID: productID, ShopID: shopID, TargetStoreID: shopID,
		Platform: "douyin_shop", TaskType: TaskTypeDouyinDraftCreate, Status: TaskPending, PublishStatus: StatusChecking,
		Mode: PublishModeSaveAsPlatformDraft, PublishMode: PublishModeSaveAsPlatformDraft, MappingSnapshot: datatypes.JSON(mappingRaw),
		PlatformPayload: datatypes.JSON(requestRaw), Input: datatypes.JSON(snapshotRaw), OperationTaskID: &operationTaskID,
		ExecutionAttemptID: &attemptID, FrozenPayloadHash: payloadHash,
	}
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, db.Model(&operationtask.ExecutionAttempt{}).Where("id = ?", attemptID).Update("downstream_task_id", task.ID).Error)
	return &Service{DB: db, WriteControl: &productioncontrolp10.Service{}, OperationResults: &recordingOperationResultSink{}}, task, frozen
}

type recordingOperationResultSink struct{}

func (*recordingOperationResultSink) MarkRunning(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (*recordingOperationResultSink) MarkSucceeded(context.Context, uuid.UUID, uuid.UUID, string, string, datatypes.JSON) error {
	return nil
}
func (*recordingOperationResultSink) MarkFailed(context.Context, uuid.UUID, uuid.UUID, string, string, bool, bool, datatypes.JSON) error {
	return nil
}

func TestValidateProductionDouyinTaskAcceptsBoundFrozenTask(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	svc, task, _ := seedBoundProductionDouyinTask(t, db)
	snap, build, err := svc.validateProductionDouyinTask(context.Background(), &task)
	require.NoError(t, err)
	require.Equal(t, *task.ExecutionAttemptID, *snap.ExecutionAttemptID)
	require.Equal(t, task.ProductID.String(), build.APIReq.OuterProductID)
}

func TestValidateProductionDouyinTaskRejectsFrozenCopiesAndHashesTampering(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tamper func(*gorm.DB, *ProductPublishTask, operationtask.FrozenDouyinDraft)
	}{
		{name: "platform payload", tamper: func(_ *gorm.DB, task *ProductPublishTask, _ operationtask.FrozenDouyinDraft) {
			task.PlatformPayload = datatypes.JSON([]byte(`{"name":"tampered"}`))
		}},
		{name: "mapping snapshot", tamper: func(_ *gorm.DB, task *ProductPublishTask, _ operationtask.FrozenDouyinDraft) {
			task.MappingSnapshot = datatypes.JSON([]byte(`{"platform":"douyin_shop"}`))
		}},
		{name: "frozen payload hash", tamper: func(_ *gorm.DB, task *ProductPublishTask, _ operationtask.FrozenDouyinDraft) {
			task.FrozenPayloadHash = strings.Repeat("0", 64)
		}},
		{name: "database draft payload", tamper: func(db *gorm.DB, task *ProductPublishTask, frozen operationtask.FrozenDouyinDraft) {
			frozen.Review = json.RawMessage(`{"name":"tampered"}`)
			raw, err := json.Marshal(frozen)
			require.NoError(t, err)
			require.NoError(t, db.Model(&operationtask.PlatformDraft{}).Where("operation_task_id = ?", *task.OperationTaskID).Update("payload", datatypes.JSON(raw)).Error)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newDouyinPublishTestDB(t)
			svc, task, frozen := seedBoundProductionDouyinTask(t, db)
			tc.tamper(db, &task, frozen)
			_, _, err := svc.validateProductionDouyinTask(context.Background(), &task)
			require.ErrorIs(t, err, ErrDouyinOperationTaskRequired)
		})
	}
}

func TestProcessDouyinDraftTaskRejectsUnboundLegacyTaskBeforePlatformAccess(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	task := ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 101, ProductID: uuid.New(), ShopID: uuid.New(), TargetStoreID: uuid.New(),
		Platform: "douyin_shop", TaskType: TaskTypeDouyinDraftCreate, Status: TaskPending, PublishStatus: StatusChecking,
		Mode: PublishModeSaveAsPlatformDraft, PublishMode: PublishModeSaveAsPlatformDraft,
	}
	require.NoError(t, db.Create(&task).Error)
	svc := &Service{DB: db, WriteControl: &productioncontrolp10.Service{}, OperationResults: &recordingOperationResultSink{}}
	err := svc.ProcessQueuedTask(context.Background(), task.ID, "test-worker")
	require.ErrorIs(t, err, ErrDouyinOperationTaskRequired)

	var stored ProductPublishTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.Equal(t, TaskFailed, stored.Status)
	require.Equal(t, ErrorDouyinOperationTaskRequired, stored.ErrorCode)
	require.False(t, stored.Retryable)
}

func TestUnknownDouyinResultNeverBecomesRetryable(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	task := ProductPublishTask{
		HardDeleteBase: model.HardDeleteBase{ID: uuid.New()}, TenantID: 101, ProductID: uuid.New(), ShopID: uuid.New(), TargetStoreID: uuid.New(),
		Platform: "douyin_shop", TaskType: TaskTypeDouyinDraftCreate, Status: TaskRunning, PublishStatus: StatusCreatingPlatformDraft, Retryable: true,
	}
	require.NoError(t, db.Create(&task).Error)
	svc := &Service{DB: db}
	svc.markDouyinStale(context.Background(), task.ID, platformdouyin.CodeDouyinUnknownResult, platformdouyin.RecoveryResultUnknown, nil)

	var stored ProductPublishTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.Equal(t, TaskFailed, stored.Status)
	require.Equal(t, platformdouyin.CodeDouyinUnknownResult, stored.ErrorCode)
	require.False(t, stored.Retryable)
}

func TestRecoverDouyinDraftStaleRejectsQueuedExecutionBeforePlatformAccess(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	svc, task, _ := seedBoundProductionDouyinTask(t, db)
	svc.Shops = nil

	err := svc.RecoverDouyinDraftStale(context.Background(), task.ID)
	require.ErrorIs(t, err, ErrDouyinRecoveryNotAllowed)

	var stored ProductPublishTask
	require.NoError(t, db.First(&stored, "id = ?", task.ID).Error)
	require.Equal(t, TaskPending, stored.Status)
}

func TestLegacyDouyinCreateServiceAlwaysRejectsWithoutWrites(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	svc := &Service{DB: db}
	_, err := svc.CreateDouyinDraftTask(testGinContext(), uuid.New(), DouyinCreateDraftBody{ShopID: uuid.NewString(), PublishMode: PublishModeSaveAsPlatformDraft}, nil)
	require.ErrorIs(t, err, ErrDouyinOperationTaskRequired)
	for _, modelValue := range []any{&ProductPublishTask{}, &ProductPublication{}, &ProductPublicationSKU{}} {
		var count int64
		require.NoError(t, db.Model(modelValue).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestBuildDouyinDraftSnapshotFreezesMappedContent(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	ctx := context.Background()
	tenantID := int64(101)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&product.Product{
		TenantID: tenantID, Source: "manual", Title: "Source title", Status: product.StatusReady,
	}).Error)
	var productRow product.Product
	require.NoError(t, db.Where("tenant_id = ?", tenantID).First(&productRow).Error)
	productID := productRow.ID
	require.NoError(t, db.Create(&shop.Shop{
		TenantID: tenantID, Platform: "douyin_shop", ShopName: "Authorized shop",
		Status: shop.StatusActive, AuthStatus: shop.AuthAuthorized,
	}).Error)
	var shopRow shop.Shop
	require.NoError(t, db.Where("tenant_id = ?", tenantID).First(&shopRow).Error)
	shopID := shopRow.ID
	images, err := json.Marshal(map[string]any{
		"mainImages": []map[string]any{{
			"platformImageId": "image-1", "platformImageUrl": "https://p3-aio.ecombdimg.com/obj/frozen.jpg", "uploadStatus": "uploaded",
		}},
		"detailImages": []any{},
	})
	require.NoError(t, err)
	skus, err := json.Marshal([]product.DouyinDraftSKU{{
		LocalSkuID: uuid.NewString(), Name: "Default", Price: 88.8, Stock: ptrInt(3),
	}})
	require.NoError(t, err)
	require.NoError(t, db.Create(&product.ProductPlatformPublishConfig{
		ProductID: productID, Platform: "douyin_shop", ShopID: &shopID, CategoryID: "20219",
		MappedTitle: "Frozen title", MappedDescription: "Frozen description", MappedImages: datatypes.JSON(images),
		MappedSKUs: datatypes.JSON(skus), LastMappedAt: &now,
	}).Error)

	svc := &Service{DB: db}
	raw, err := svc.BuildDouyinDraftSnapshot(ctx, tenantID, uuid.New(), operationtask.DouyinDraftIntent{
		SchemaVersion: operationtask.DouyinDraftSchemaVersion, ProductID: productID, ShopID: shopID,
		PublishMode: operationtask.DouyinDraftPublishMode,
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&product.ProductPlatformPublishConfig{}).
		Where("product_id = ? AND platform = ?", productID, "douyin_shop").
		Update("mapped_title", "Changed after freeze").Error)

	frozen, err := operationtask.ParseFrozenDouyinDraft(raw)
	require.NoError(t, err)
	var request platformdouyin.CreateProductDraftRequest
	require.NoError(t, json.Unmarshal(frozen.Request, &request))
	require.Equal(t, "Frozen title", request.Name)
	var mapping product.DouyinDraftMapping
	require.NoError(t, json.Unmarshal(frozen.MappingSnapshot, &mapping))
	require.Equal(t, "Frozen title", mapping.Title)
}

func TestBuildDouyinProductPayloadRejectsUnuploadedImages(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	pid := uuid.New()
	images, _ := json.Marshal(map[string]any{
		"mainImages": []map[string]any{{
			"url": "https://img.example.com/a.jpg", "uploadStatus": "pending",
		}},
		"detailImages": []any{},
	})
	skus, _ := json.Marshal([]product.DouyinDraftSKU{{
		LocalSkuID: uuid.NewString(), Name: "Default", Price: 99, Stock: ptrInt(10),
	}})
	price, _ := json.Marshal(product.DouyinDraftPrice{Currency: "CNY", Min: ptrFloat(99)})
	stock, _ := json.Marshal(product.DouyinDraftStock{Total: ptrInt(10)})
	if err := db.Create(&product.ProductPlatformPublishConfig{
		ProductID: pid, Platform: "douyin_shop", CategoryID: "12345",
		MappedTitle: "Test Product", MappedImages: datatypes.JSON(images),
		MappedSKUs: datatypes.JSON(skus), MappedPrice: datatypes.JSON(price), MappedStock: datatypes.JSON(stock),
	}).Error; err != nil {
		t.Fatal(err)
	}
	res, err := BuildDouyinProductPayload(context.Background(), db, pid, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected errors for unuploaded main image")
	}
	found := false
	for _, e := range res.Errors {
		if e.Code == product.DouyinMainImageNotUploaded {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DOUYIN_MAIN_IMAGE_NOT_UPLOADED, got %+v", res.Errors)
	}
}

func TestBuildDouyinProductPayloadUsesUploadedImagesNotRaw(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	pid := uuid.New()
	images, _ := json.Marshal(map[string]any{
		"mainImages": []map[string]any{{
			"platformImageUrl": "https://p3-aio.ecombdimg.com/obj/test.jpg",
			"uploadStatus":     "uploaded",
		}},
		"detailImages": []any{},
	})
	skus, _ := json.Marshal([]product.DouyinDraftSKU{{
		LocalSkuID: uuid.NewString(), Name: "Red", Price: 88.8, Stock: ptrInt(3), Attrs: map[string]any{"颜色": "红"},
	}})
	price, _ := json.Marshal(product.DouyinDraftPrice{Currency: "CNY", Min: ptrFloat(88.8)})
	stock, _ := json.Marshal(product.DouyinDraftStock{Total: ptrInt(3)})
	attrs, _ := json.Marshal(map[string]any{"405": "27664"})
	if err := db.Create(&product.ProductPlatformPublishConfig{
		ProductID: pid, Platform: "douyin_shop", CategoryID: "20219",
		MappedTitle: "抖店测试商品", MappedImages: datatypes.JSON(images),
		MappedSKUs: datatypes.JSON(skus), MappedPrice: datatypes.JSON(price), MappedStock: datatypes.JSON(stock),
		PlatformAttributes: datatypes.JSON(attrs),
	}).Error; err != nil {
		t.Fatal(err)
	}
	res, err := BuildDouyinProductPayload(context.Background(), db, pid, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
	if res.Payload == nil || !containsStr(res.Payload.Pic, "ecombdimg.com") {
		t.Fatalf("expected uploaded platform image url in pic: %+v", res.Payload)
	}
	if len(res.APIReq.SpecPricesV2) != 1 || res.APIReq.SpecPricesV2[0]["price"] != int64(8880) {
		t.Fatalf("expected price in fen: %+v", res.APIReq.SpecPricesV2)
	}
}

func TestBuildDouyinProductPayloadRejectsInvalidSKUPrice(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	pid := uuid.New()
	images, _ := json.Marshal(map[string]any{
		"mainImages": []map[string]any{{
			"platformImageUrl": "https://p3-aio.ecombdimg.com/obj/test.jpg", "uploadStatus": "uploaded",
		}},
	})
	skus, _ := json.Marshal([]product.DouyinDraftSKU{{LocalSkuID: uuid.NewString(), Name: "Bad", Price: 0, Stock: ptrInt(1)}})
	price, _ := json.Marshal(product.DouyinDraftPrice{Currency: "CNY", Min: ptrFloat(0)})
	if err := db.Create(&product.ProductPlatformPublishConfig{
		ProductID: pid, Platform: "douyin_shop", CategoryID: "20219",
		MappedTitle: "Test", MappedImages: datatypes.JSON(images),
		MappedSKUs: datatypes.JSON(skus), MappedPrice: datatypes.JSON(price),
	}).Error; err != nil {
		t.Fatal(err)
	}
	res, err := BuildDouyinProductPayload(context.Background(), db, pid, "")
	if err != nil {
		t.Fatal(err)
	}
	if !issueHasCode(res.Errors, product.DouyinSKUPriceInvalid) {
		t.Fatalf("expected invalid sku price error, got %+v", res.Errors)
	}
}

func TestBuildDouyinProductPayloadMissingMappingConfig(t *testing.T) {
	db := newDouyinPublishTestDB(t)
	_, err := BuildDouyinProductPayload(context.Background(), db, uuid.New(), "")
	if err == nil {
		t.Fatal("expected error when mapping config missing")
	}
}

func ptrInt(v int) *int           { return &v }
func ptrFloat(v float64) *float64 { return &v }

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func issueHasCode(items []product.DouyinMappingIssue, code string) bool {
	for _, it := range items {
		if it.Code == code {
			return true
		}
	}
	return false
}
