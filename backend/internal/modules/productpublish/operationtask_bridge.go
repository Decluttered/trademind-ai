package productpublish

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationtask"
	"github.com/trademind-ai/trademind/backend/internal/modules/product"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) BuildDouyinDraftSnapshot(ctx context.Context, tenantID int64, actorID uuid.UUID, intent operationtask.DouyinDraftIntent) (datatypes.JSON, error) {
	if s == nil || s.DB == nil || tenantID <= 0 || actorID == uuid.Nil || intent.ProductID == uuid.Nil || intent.ShopID == uuid.Nil {
		return nil, operationtask.ErrValidation
	}
	var snapshot datatypes.JSON
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var productRow product.Product
		if err := tx.Where("id = ? AND tenant_id = ?", intent.ProductID, tenantID).First(&productRow).Error; err != nil {
			return operationtask.ErrNotFound
		}
		var shopRow shop.Shop
		if err := tx.Where("id = ? AND tenant_id = ? AND platform = ? AND auth_status = ?", intent.ShopID, tenantID, "douyin_shop", shop.AuthAuthorized).First(&shopRow).Error; err != nil {
			return operationtask.ErrNotFound
		}
		var cfg product.ProductPlatformPublishConfig
		if err := tx.Where("product_id = ? AND platform = ?", intent.ProductID, "douyin_shop").First(&cfg).Error; err != nil {
			return operationtask.ErrValidation
		}
		if cfg.ShopID == nil || *cfg.ShopID != intent.ShopID || cfg.LastMappedAt == nil {
			return operationtask.ErrValidation
		}
		build, err := BuildDouyinProductPayload(ctx, tx, intent.ProductID, cfg.ID.String())
		if err != nil || build == nil || build.Payload == nil || len(build.Errors) > 0 || len(build.APIReq.SpecPricesV2) < 1 {
			return operationtask.ErrValidation
		}
		build.APIReq.PublishConfig = s.frozenDouyinPublishConfig(ctx, tx, tenantID)
		mapping := product.DouyinDraftMappingFromConfig(cfg)
		requestRaw, err := json.Marshal(build.APIReq)
		if err != nil {
			return err
		}
		reviewRaw, err := json.Marshal(sanitizeDouyinPayloadForDisplay(build.Payload))
		if err != nil {
			return err
		}
		mappingRaw, err := json.Marshal(mapping)
		if err != nil {
			return err
		}
		mappingHash, err := operationtask.ComputePayloadHash(mappingRaw)
		if err != nil {
			return err
		}
		frozen := operationtask.FrozenDouyinDraft{
			SchemaVersion: operationtask.DouyinDraftSchemaVersion, ProductID: intent.ProductID, ShopID: intent.ShopID,
			PublishMode: operationtask.DouyinDraftPublishMode, SKUCount: len(build.APIReq.SpecPricesV2), Request: requestRaw,
			Review: reviewRaw, MappingSnapshot: mappingRaw, MappingHash: mappingHash,
		}
		raw, err := json.Marshal(frozen)
		if err != nil {
			return err
		}
		snapshot = datatypes.JSON(raw)
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return snapshot, err
}

func (s *Service) frozenDouyinPublishConfig(ctx context.Context, db *gorm.DB, tenantID int64) map[string]string {
	out := map[string]string{}
	if s == nil || s.Settings == nil || db == nil {
		return out
	}
	settingsReader := *s.Settings
	settingsReader.DB = db
	values, err := settingsReader.PlainByGroup(ctx, tenantID, "platform_publish_douyin_shop")
	if err != nil || len(values) == 0 {
		values, _ = settingsReader.PlainByGroup(ctx, 0, "platform_publish_douyin_shop")
	}
	for _, key := range []string{"delivery_delay_day", "default_mobile", "standard_brand_id", "after_sale_service"} {
		if value := strings.TrimSpace(values[key]); value != "" {
			out[key] = value
		}
	}
	return out
}

func (s *Service) CreateFrozenDouyinDraftTask(ctx context.Context, tx *gorm.DB, in operationtask.ProductionDownstreamInput) (uuid.UUID, error) {
	if s == nil || tx == nil || in.TenantID <= 0 || in.OperationTaskID == uuid.Nil || in.ExecutionAttemptID == uuid.Nil || in.ActorID == uuid.Nil {
		return uuid.Nil, operationtask.ErrValidation
	}
	var request platformdouyin.CreateProductDraftRequest
	var mapping product.DouyinDraftMapping
	if err := json.Unmarshal(in.FrozenDraft.Request, &request); err != nil {
		return uuid.Nil, operationtask.ErrValidation
	}
	if err := json.Unmarshal(in.FrozenDraft.MappingSnapshot, &mapping); err != nil {
		return uuid.Nil, operationtask.ErrValidation
	}
	mappingHash, err := operationtask.ComputePayloadHash(in.FrozenDraft.MappingSnapshot)
	if err != nil || mappingHash != strings.ToLower(strings.TrimSpace(in.FrozenDraft.MappingHash)) {
		return uuid.Nil, operationtask.ErrValidation
	}
	frozenRaw, err := json.Marshal(in.FrozenDraft)
	if err != nil {
		return uuid.Nil, operationtask.ErrValidation
	}
	payloadHash, err := operationtask.ComputePayloadHash(frozenRaw)
	if err != nil || payloadHash != strings.ToLower(strings.TrimSpace(in.PayloadHash)) {
		return uuid.Nil, operationtask.ErrDraftBindingConflict
	}
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.OuterProductID) != in.FrozenDraft.ProductID.String() || len(request.SpecPricesV2) != in.FrozenDraft.SKUCount || mapping.ProductID != in.FrozenDraft.ProductID.String() || mapping.ShopID != in.FrozenDraft.ShopID.String() {
		return uuid.Nil, operationtask.ErrValidation
	}
	var activeCount int64
	if err := tx.Model(&ProductPublishTask{}).Where("tenant_id = ? AND product_id = ? AND shop_id = ? AND platform = ? AND status IN ?", in.TenantID, in.FrozenDraft.ProductID, in.FrozenDraft.ShopID, "douyin_shop", []string{TaskPending, TaskRunning}).Count(&activeCount).Error; err != nil {
		return uuid.Nil, err
	}
	if activeCount > 0 {
		return uuid.Nil, operationtask.ErrExecutionInProgress
	}
	var existing ProductPublication
	err = tx.Where("product_id = ? AND shop_id = ? AND platform = ? AND external_product_id <> ''", in.FrozenDraft.ProductID, in.FrozenDraft.ShopID, "douyin_shop").First(&existing).Error
	if err == nil {
		return uuid.Nil, operationtask.ErrStateConflict
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}

	publication := ProductPublication{ProductID: in.FrozenDraft.ProductID, ShopID: in.FrozenDraft.ShopID, Platform: "douyin_shop", Status: StatusDraft, PublishStatus: StatusChecking, Title: request.Name, Currency: "CNY", CreatedBy: &in.ActorID}
	if err := tx.Create(&publication).Error; err != nil {
		return uuid.Nil, err
	}
	snapshot := douyinDraftSnapshot{
		PublicationID: publication.ID, PublishMode: PublishModeSaveAsPlatformDraft, MappingHash: in.FrozenDraft.MappingHash,
		Mapping: map[string]any{}, FrozenRequest: &request, FrozenMapping: &mapping,
		ExecutionAttemptID: &in.ExecutionAttemptID, OperationTaskID: &in.OperationTaskID, FrozenPayloadHash: in.PayloadHash,
	}
	snapshotRaw, err := json.Marshal(snapshot)
	if err != nil {
		return uuid.Nil, err
	}
	platformRaw, err := json.Marshal(request)
	if err != nil {
		return uuid.Nil, err
	}
	mappingRaw, err := json.Marshal(mapping)
	if err != nil {
		return uuid.Nil, err
	}
	task := ProductPublishTask{
		TenantID: in.TenantID, ProductID: in.FrozenDraft.ProductID, ShopID: in.FrozenDraft.ShopID, TargetStoreID: in.FrozenDraft.ShopID,
		Platform: "douyin_shop", TaskType: TaskTypeDouyinDraftCreate, Status: TaskPending, PublishStatus: StatusChecking,
		Mode: PublishModeSaveAsPlatformDraft, PublishMode: PublishModeSaveAsPlatformDraft, Title: request.Name, Description: request.Description,
		MappingSnapshot: datatypes.JSON(mappingRaw), PlatformPayload: datatypes.JSON(platformRaw), Input: datatypes.JSON(snapshotRaw),
		CreatedBy: &in.ActorID, RequestID: in.RequestID, OperationTaskID: &in.OperationTaskID, ExecutionAttemptID: &in.ExecutionAttemptID, FrozenPayloadHash: in.PayloadHash,
	}
	if err := tx.Create(&task).Error; err != nil {
		return uuid.Nil, err
	}
	if err := tx.Model(&ProductPublication{}).Where("id = ?", publication.ID).Update("publish_task_id", task.ID).Error; err != nil {
		return uuid.Nil, err
	}
	return task.ID, nil
}

func (s *Service) validateProductionDouyinTask(ctx context.Context, task *ProductPublishTask) (douyinDraftSnapshot, *DouyinPayloadBuildResult, error) {
	var empty douyinDraftSnapshot
	if s == nil || s.DB == nil || s.WriteControl == nil || s.OperationResults == nil || task == nil || task.ID == uuid.Nil || task.TenantID <= 0 || task.ProductID == uuid.Nil || task.ShopID == uuid.Nil ||
		!strings.EqualFold(strings.TrimSpace(task.Platform), "douyin_shop") || task.TaskType != TaskTypeDouyinDraftCreate || task.PublishMode != PublishModeSaveAsPlatformDraft ||
		task.OperationTaskID == nil || *task.OperationTaskID == uuid.Nil || task.ExecutionAttemptID == nil || *task.ExecutionAttemptID == uuid.Nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	snap, ok := parseDouyinDraftSnapshot(task.Input)
	if !ok || snap.OperationTaskID == nil || *snap.OperationTaskID != *task.OperationTaskID || snap.ExecutionAttemptID == nil || *snap.ExecutionAttemptID != *task.ExecutionAttemptID ||
		snap.FrozenRequest == nil || snap.FrozenMapping == nil || snap.PublicationID == uuid.Nil || snap.PublishMode != PublishModeSaveAsPlatformDraft ||
		strings.ToLower(strings.TrimSpace(snap.FrozenPayloadHash)) != strings.ToLower(strings.TrimSpace(task.FrozenPayloadHash)) {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	request := snap.FrozenRequest
	mapping := snap.FrozenMapping
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.OuterProductID) != task.ProductID.String() || mapping.ProductID != task.ProductID.String() || mapping.ShopID != task.ShopID.String() || len(request.SpecPricesV2) < 1 {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	platformPayloadHash, err := operationtask.ComputePayloadHash(task.PlatformPayload)
	if err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	requestRaw, err := json.Marshal(request)
	if err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	requestHash, err := operationtask.ComputePayloadHash(requestRaw)
	if err != nil || requestHash != platformPayloadHash {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	mappingRaw, err := json.Marshal(mapping)
	if err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	mappingHash, err := operationtask.ComputePayloadHash(mappingRaw)
	taskMappingHash, taskMappingErr := operationtask.ComputePayloadHash(task.MappingSnapshot)
	if err != nil || taskMappingErr != nil || mappingHash != taskMappingHash || mappingHash != strings.ToLower(strings.TrimSpace(snap.MappingHash)) {
		return empty, nil, ErrDouyinOperationTaskRequired
	}

	var attempt operationtask.ExecutionAttempt
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ? AND operation_task_id = ? AND downstream_task_id = ?", *task.ExecutionAttemptID, task.TenantID, *task.OperationTaskID, task.ID).First(&attempt).Error; err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	if attempt.AdapterMode != operationtask.AdapterModeProductionDraft || attempt.Platform != operationtask.PlatformDouyin ||
		(attempt.Status != operationtask.ExecutionAttemptStatusQueued && attempt.Status != operationtask.ExecutionAttemptStatusRunning && attempt.Status != operationtask.ExecutionAttemptStatusResultUnknown) {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	var operationTask operationtask.OperationTask
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ?", *task.OperationTaskID, task.TenantID).First(&operationTask).Error; err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	if operationTask.TaskType != operationtask.OperationTaskTypeProductPublish || operationTask.Platform != operationtask.PlatformDouyin ||
		(operationTask.Status != operationtask.OperationTaskStatusExecutionQueued && operationTask.Status != operationtask.OperationTaskStatusExecuting && operationTask.Status != operationtask.OperationTaskStatusResultUnknown) {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	var draft operationtask.PlatformDraft
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ? AND operation_task_id = ?", attempt.PlatformDraftID, task.TenantID, *task.OperationTaskID).First(&draft).Error; err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	var approval operationtask.ApprovalRecord
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ? AND operation_task_id = ? AND platform_draft_id = ?", attempt.ApprovalRecordID, task.TenantID, *task.OperationTaskID, draft.ID).First(&approval).Error; err != nil {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	if approval.Decision != operationtask.ApprovalDecisionApproved || approval.DraftVersion != draft.DraftVersion || approval.DraftPayloadHash != draft.PayloadHash ||
		attempt.ApprovedDraftVersion != approval.DraftVersion || attempt.ExecutedDraftVersion != draft.DraftVersion {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	draftHash, err := operationtask.ComputePayloadHash(draft.Payload)
	if err != nil || draftHash != draft.PayloadHash || draft.PayloadHash != task.FrozenPayloadHash || attempt.ExecutedDraftPayloadHash != draft.PayloadHash || attempt.ApprovedDraftPayloadHash != draft.PayloadHash {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	frozen, err := operationtask.ParseFrozenDouyinDraft(draft.Payload)
	if err != nil || frozen.ProductID != task.ProductID || frozen.ShopID != task.ShopID || frozen.SKUCount != len(request.SpecPricesV2) || strings.ToLower(strings.TrimSpace(frozen.MappingHash)) != mappingHash {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	frozenRequestHash, requestHashErr := operationtask.ComputePayloadHash(frozen.Request)
	frozenMappingHash, frozenMappingErr := operationtask.ComputePayloadHash(frozen.MappingSnapshot)
	if requestHashErr != nil || frozenMappingErr != nil || frozenRequestHash != requestHash || frozenMappingHash != mappingHash {
		return empty, nil, ErrDouyinOperationTaskRequired
	}
	return snap, &DouyinPayloadBuildResult{Payload: frozenRequestDisplayPayload(*request), APIReq: *request}, nil
}

func (s *Service) EnqueueProductionTask(ctx context.Context, taskID uuid.UUID) error {
	if s == nil || !s.QueueEnabled || s.Redis == nil || s.Redis.Client == nil || taskID == uuid.Nil {
		return fmt.Errorf("production product publish queue unavailable")
	}
	return s.enqueue(ctx, taskID)
}
