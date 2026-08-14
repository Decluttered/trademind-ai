package inventoryread

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/modules/inventorysync"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RunInput struct {
	TenantID         int64
	ActorID          uuid.UUID
	ShopID           uuid.UUID
	RequestID        string
	IdempotencyHash  string
	SourceRunID      uuid.UUID
	ExpectedRevision int
}

type RunResult struct {
	RunID              uuid.UUID `json:"runId"`
	Status             string    `json:"status"`
	SnapshotCount      int       `json:"snapshotCount"`
	CalibrationCount   int       `json:"calibrationCount"`
	ManualReviewCount  int       `json:"manualReviewCount"`
	RequestID          string    `json:"requestId"`
	ProviderMode       string    `json:"providerMode"`
	AutomaticRetryUsed bool      `json:"automaticRetryUsed"`
}

type ManualReadService struct {
	DB          *gorm.DB
	Provider    inventorysync.InventoryProvider
	Calibration *inventorysync.SKUBindingCalibrationService
	Audit       *inventorysync.InventorySyncAuditService
	Metrics     *metrics.Catalog
	Environment string
	PageSize    int
	MaxPages    int
	Now         func() time.Time
}

func NewManualReadService(db *gorm.DB, provider inventorysync.InventoryProvider, catalog *metrics.Catalog, pageSize, maxPages int) *ManualReadService {
	policy, _ := inventorysync.NewCalibrationThresholdPolicy(inventorysync.CalibrationThresholdConfig{HighConfidenceThreshold: 9500, AutoConfirmationEnabled: false, PolicyVersion: inventorysync.ThresholdPolicyVersionV1})
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if maxPages < 1 || maxPages > 100 {
		maxPages = 100
	}
	return &ManualReadService{DB: db, Provider: provider, Calibration: inventorysync.NewSKUBindingCalibrationService(db, inventorysync.NewGORMLocalSKUCandidateProvider(db), policy), Audit: inventorysync.NewInventorySyncAuditService(db), Metrics: catalog, Environment: "unknown", PageSize: pageSize, MaxPages: maxPages}
}

func (s *ManualReadService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *ManualReadService) Run(ctx context.Context, input RunInput) (*RunResult, error) {
	if s == nil || s.DB == nil || s.Provider == nil || s.Calibration == nil || input.TenantID <= 0 || input.ActorID == uuid.Nil || input.ShopID == uuid.Nil || strings.TrimSpace(input.RequestID) == "" || len(input.IdempotencyHash) != 64 {
		return nil, &ProviderError{Code: ErrorInvalidRequest}
	}
	var shopRow shop.Shop
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND id = ? AND platform IN ?", input.TenantID, input.ShopID, []string{"douyin", "douyin_shop"}).First(&shopRow).Error; err != nil {
		return nil, &ProviderError{Code: ErrorUnauthorized, Cause: err}
	}
	if input.SourceRunID != uuid.Nil {
		var source inventorysync.InventorySyncRun
		if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND id = ?", input.TenantID, input.SourceRunID).First(&source).Error; err != nil {
			return nil, &ProviderError{Code: ErrorInvalidRequest, Cause: err}
		}
		if source.Revision != input.ExpectedRevision || (source.Status != inventorysync.InventorySyncRunStatusFailed && source.Status != inventorysync.InventorySyncRunStatusCancelled) {
			return nil, &ProviderError{Code: ErrorInvalidRequest, Cause: errors.New("manual rerun requires failed source revision")}
		}
	}
	fingerprint := runFingerprint(input)
	var existing inventorysync.InventorySyncRun
	if err := s.DB.WithContext(ctx).Where("tenant_id = ? AND idempotency_key_hash = ?", input.TenantID, input.IdempotencyHash).First(&existing).Error; err == nil {
		if existing.InputFingerprint != fingerprint {
			return nil, &ProviderError{Code: ErrorInvalidRequest, Cause: errors.New("idempotency payload conflict")}
		}
		return runResult(existing), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	started := s.now()
	run := inventorysync.InventorySyncRun{TenantID: input.TenantID, ShopConnectionID: input.ShopID, Platform: inventorysync.PlatformDouyin, ProviderMode: ProviderModeRealReadOnly, Status: inventorysync.InventorySyncRunStatusRunning, Cursor: datatypes.JSON([]byte(`{}`)), Checkpoint: datatypes.JSON([]byte(`{"triggerType":"manual","readOnly":true}`)), SafeErrorMetadata: datatypes.JSON([]byte(`{}`)), RequestID: input.RequestID, IdempotencyKeyHash: input.IdempotencyHash, InputFingerprint: fingerprint, Revision: 1, StartedAt: &started}
	if input.SourceRunID != uuid.Nil {
		run.RerunOfRunID = &input.SourceRunID
		run.RerunSourceRevision = input.ExpectedRevision
	}
	if err := s.DB.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}
	s.Metrics.ObserveProductionCapabilities(s.Environment, "douyin", "inventory_read", "started", false, false, false, true, false, -1)
	_ = s.Audit.Write(ctx, inventorysync.InventorySyncAuditEvent{TenantID: input.TenantID, ActorID: input.ActorID, Action: "inventory_sync.p10_read_started", Resource: "inventory_sync", ResourceID: run.ID.String(), ShopID: input.ShopID, Platform: inventorysync.PlatformDouyin, Permission: adminperm.PermInventorySyncRun, Status: "success", RequestID: input.RequestID, Metadata: map[string]any{"providerMode": ProviderModeRealReadOnly, "readOnly": true, "automaticRetry": false}})
	pageSize := s.PageSize
	cursor := datatypes.JSON([]byte(`{}`))
	snapshotCount, calibrationCount, manualCount, pages, networkCalls := 0, 0, 0, 0, 0
	for {
		pageStarted := time.Now()
		providerCtx := withInternalRequestID(ctx, input.RequestID)
		page, err := s.Provider.FetchInventoryPage(providerCtx, inventorysync.InventoryFetchRequest{TenantID: input.TenantID, ShopConnectionID: input.ShopID.String(), Platform: inventorysync.PlatformDouyin, ProviderMode: ProviderModeRealReadOnly, Cursor: cursor, PageSize: pageSize, MaxItemsPerPage: 100})
		if err != nil {
			s.Metrics.ObserveProvider("douyin", "inventory_read", providerMetricResult(err), safeProviderErrorClass(err), time.Since(pageStarted), isTimeout(err))
			return s.failRun(ctx, run, input, err, pages, snapshotCount)
		}
		s.Metrics.ObserveProvider("douyin", "inventory_read", "success", "none", time.Since(pageStarted), false)
		pages++
		networkCalls += page.NetworkCalls
		if snapshotCount+len(page.Items) > 100 {
			return s.failRun(ctx, run, input, productioncontrolScopeError(), pages, snapshotCount)
		}
		for _, item := range page.Items {
			snapshot, err := makeSnapshot(run, item, s.now())
			if err != nil {
				return s.failRun(ctx, run, input, err, pages, snapshotCount)
			}
			if err := s.DB.WithContext(ctx).Create(&snapshot).Error; err != nil {
				return s.failRun(ctx, run, input, err, pages, snapshotCount)
			}
			snapshotCount++
			calibrated, err := s.Calibration.CalibrateSnapshotItem(ctx, input.TenantID, run.ID, snapshot.ID)
			if err != nil {
				return s.failRun(ctx, run, input, err, pages, snapshotCount)
			}
			calibrationCount += len(calibrated.Candidates)
			if calibrated.ManualBindingRequest != nil {
				manualCount++
			}
		}
		cursor = page.NextCursor
		if !page.HasMore {
			break
		}
		if page.HasMore && pages >= s.MaxPages {
			return s.failRun(ctx, run, input, productioncontrolScopeError(), pages, snapshotCount)
		}
	}
	finished := s.now()
	checkpoint, _ := json.Marshal(map[string]any{"triggerType": "manual", "readOnly": true, "pagesProcessed": pages, "totalRecordCount": snapshotCount, "manualBindingRequestCount": manualCount, "providerNetworkCalls": networkCalls, "automaticRetry": false})
	updates := map[string]any{"status": inventorysync.InventorySyncRunStatusSucceeded, "cursor": cursor, "checkpoint": datatypes.JSON(checkpoint), "snapshot_count": snapshotCount, "calibration_count": calibrationCount, "manual_request_count": manualCount, "finished_at": finished, "revision": 2, "updated_at": finished}
	if err := s.DB.WithContext(ctx).Model(&inventorysync.InventorySyncRun{}).Where("tenant_id = ? AND id = ? AND revision = 1", input.TenantID, run.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	run.Status, run.SnapshotCount, run.CalibrationCount, run.ManualRequestCount, run.Revision = inventorysync.InventorySyncRunStatusSucceeded, snapshotCount, calibrationCount, manualCount, 2
	_ = s.Audit.Write(ctx, inventorysync.InventorySyncAuditEvent{TenantID: input.TenantID, ActorID: input.ActorID, Action: "inventory_sync.p10_read_succeeded", Resource: "inventory_sync", ResourceID: run.ID.String(), ShopID: input.ShopID, Platform: inventorysync.PlatformDouyin, Permission: adminperm.PermInventorySyncRun, Status: "success", RequestID: input.RequestID, Metadata: map[string]any{"snapshotCount": snapshotCount, "calibrationCount": calibrationCount, "manualBindingRequestCount": manualCount, "readOnly": true}})
	s.Metrics.ObserveInventory("douyin", "read_snapshot", "run", "success", "none", 1, finished.Sub(started))
	s.Metrics.ObserveProductionCapabilities(s.Environment, "douyin", "inventory_read", "success", false, false, false, false, false, manualCount)
	return runResult(run), nil
}

func (s *ManualReadService) failRun(ctx context.Context, run inventorysync.InventorySyncRun, input RunInput, cause error, pages, snapshots int) (*RunResult, error) {
	finished := s.now()
	code := safeProviderErrorClass(cause)
	retryAfterSeconds := int64(0)
	providerRequestID := ""
	var providerErr *ProviderError
	if errors.As(cause, &providerErr) && providerErr != nil {
		retryAfterSeconds = int64(providerErr.RetryAfter / time.Second)
		providerRequestID = providerErr.ProviderRequestID
	}
	meta, _ := json.Marshal(map[string]any{"errorCode": code, "safeMessage": code, "providerRequestId": providerRequestID, "pagesProcessed": pages, "snapshotCount": snapshots, "retryAfterSeconds": retryAfterSeconds, "automaticRetry": false})
	_ = s.DB.WithContext(ctx).Model(&inventorysync.InventorySyncRun{}).Where("tenant_id = ? AND id = ?", input.TenantID, run.ID).Updates(map[string]any{"status": inventorysync.InventorySyncRunStatusFailed, "safe_error_metadata": datatypes.JSON(meta), "finished_at": finished, "revision": 2, "updated_at": finished}).Error
	_ = s.Audit.Write(ctx, inventorysync.InventorySyncAuditEvent{TenantID: input.TenantID, ActorID: input.ActorID, Action: "inventory_sync.p10_read_failed", Resource: "inventory_sync", ResourceID: run.ID.String(), ShopID: input.ShopID, Platform: inventorysync.PlatformDouyin, Permission: adminperm.PermInventorySyncRun, Status: "failed", RequestID: input.RequestID, Metadata: map[string]any{"errorCode": code, "safeMessage": code, "readOnly": true, "automaticRetry": false}})
	s.Metrics.ObserveInventory("douyin", "read_snapshot", "run", "failure", code, 1, finished.Sub(*run.StartedAt))
	s.Metrics.ObserveProductionCapabilities(s.Environment, "douyin", "inventory_read", code, true, false, false, false, true, -1)
	return nil, cause
}

func makeSnapshot(run inventorysync.InventorySyncRun, item inventorysync.InventoryProviderItem, observed time.Time) (inventorysync.InventorySnapshotItem, error) {
	payload, err := json.Marshal(item)
	if err != nil {
		return inventorysync.InventorySnapshotItem{}, err
	}
	sum := sha256.Sum256(payload)
	meta, err := json.Marshal(item.SafeMetadata)
	if err != nil {
		return inventorysync.InventorySnapshotItem{}, err
	}
	return inventorysync.InventorySnapshotItem{TenantID: run.TenantID, InventorySyncRunID: run.ID, ShopConnectionID: run.ShopConnectionID, Platform: run.Platform, ExternalProductID: strings.TrimSpace(item.ExternalProductID), ExternalSKUID: strings.TrimSpace(item.ExternalSKUID), ExternalProductCode: strings.TrimSpace(item.ExternalProductCode), ExternalSKUCode: strings.TrimSpace(item.ExternalSKUCode), Barcode: strings.TrimSpace(item.Barcode), ProductTitle: strings.TrimSpace(item.ProductTitle), VariantTitle: strings.TrimSpace(item.VariantTitle), AvailableQuantity: item.AvailableQuantity, ReservedQuantity: item.ReservedQuantity, TotalQuantity: item.TotalQuantity, SourceUpdatedAt: item.SourceUpdatedAt, ObservedAt: observed, PayloadHash: hex.EncodeToString(sum[:]), SafeMetadata: datatypes.JSON(meta)}, nil
}

func runFingerprint(input RunInput) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%d", input.TenantID, input.ShopID, input.SourceRunID, input.ExpectedRevision)))
	return hex.EncodeToString(sum[:])
}

func runResult(run inventorysync.InventorySyncRun) *RunResult {
	return &RunResult{RunID: run.ID, Status: run.Status, SnapshotCount: run.SnapshotCount, CalibrationCount: run.CalibrationCount, ManualReviewCount: run.ManualRequestCount, RequestID: run.RequestID, ProviderMode: run.ProviderMode, AutomaticRetryUsed: false}
}

func safeProviderErrorClass(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr != nil {
		switch providerErr.Code {
		case ErrorUnauthorized:
			return "unauthorized"
		case ErrorCredentialExpired:
			return "credential_expired"
		case ErrorRateLimited:
			return "rate_limited"
		case ErrorProviderUnavailable:
			return "provider_unavailable"
		case ErrorInvalidRequest:
			return "invalid_request"
		default:
			return "provider_protocol_error"
		}
	}
	return "provider_protocol_error"
}

func isTimeout(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr != nil && providerErr.Timeout
}

func providerMetricResult(err error) string {
	if isTimeout(err) {
		return "timeout"
	}
	if safeProviderErrorClass(err) == "rate_limited" {
		return "rate_limited"
	}
	return "failure"
}

func productioncontrolScopeError() error {
	return &ProviderError{Code: ErrorInvalidRequest, Cause: errors.New("approved read scope exceeded")}
}
