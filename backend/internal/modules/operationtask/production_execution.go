package operationtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DouyinDraftSchemaVersion = "douyin_draft_v1"
	DouyinDraftPublishMode   = "save_as_platform_draft"
)

type DouyinDraftIntent struct {
	SchemaVersion string    `json:"schemaVersion"`
	ProductID     uuid.UUID `json:"productId"`
	ShopID        uuid.UUID `json:"shopId"`
	PublishMode   string    `json:"publishMode"`
}

type FrozenDouyinDraft struct {
	SchemaVersion   string          `json:"schemaVersion"`
	ProductID       uuid.UUID       `json:"productId"`
	ShopID          uuid.UUID       `json:"shopId"`
	PublishMode     string          `json:"publishMode"`
	SKUCount        int             `json:"skuCount"`
	Request         json.RawMessage `json:"request"`
	Review          json.RawMessage `json:"review"`
	MappingSnapshot json.RawMessage `json:"mappingSnapshot"`
	MappingHash     string          `json:"mappingHash"`
}

func ParseDouyinDraftIntent(raw []byte) (DouyinDraftIntent, error) {
	var out DouyinDraftIntent
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, ErrValidation
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return out, ErrValidation
	}
	out.SchemaVersion = strings.TrimSpace(strings.ToLower(out.SchemaVersion))
	out.PublishMode = strings.TrimSpace(strings.ToLower(out.PublishMode))
	if out.SchemaVersion != DouyinDraftSchemaVersion || out.ProductID == uuid.Nil || out.ShopID == uuid.Nil || out.PublishMode != DouyinDraftPublishMode {
		return out, ErrValidation
	}
	return out, nil
}

func ParseFrozenDouyinDraft(raw []byte) (FrozenDouyinDraft, error) {
	var out FrozenDouyinDraft
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, ErrValidation
	}
	if out.SchemaVersion != DouyinDraftSchemaVersion || out.ProductID == uuid.Nil || out.ShopID == uuid.Nil || out.PublishMode != DouyinDraftPublishMode || out.SKUCount < 1 || len(out.Request) == 0 || len(out.Review) == 0 || len(out.MappingSnapshot) == 0 || !json.Valid(out.Request) || !json.Valid(out.Review) || !json.Valid(out.MappingSnapshot) {
		return out, ErrValidation
	}
	return out, nil
}

func IsProductionDouyinTask(task *OperationTask, draft *PlatformDraft) bool {
	return task != nil && draft != nil && task.TaskType == OperationTaskTypeProductPublish && task.Platform == PlatformDouyin && draft.AdapterMode == AdapterModeProductionDraft
}

type ProductionSnapshotBuilder interface {
	BuildDouyinDraftSnapshot(ctx context.Context, tenantID int64, actorID uuid.UUID, intent DouyinDraftIntent) (datatypes.JSON, error)
}

type ProductionWriteAuthorizer interface {
	EvaluateWrite(ctx context.Context, tenantID int64, shopID, productID uuid.UUID, skuCount int) error
}

type ProductionDownstreamInput struct {
	TenantID           int64
	OperationTaskID    uuid.UUID
	ExecutionAttemptID uuid.UUID
	ActorID            uuid.UUID
	RequestID          string
	PayloadHash        string
	FrozenDraft        FrozenDouyinDraft
}

type ProductionDownstreamFactory interface {
	CreateFrozenDouyinDraftTask(ctx context.Context, tx *gorm.DB, in ProductionDownstreamInput) (uuid.UUID, error)
}

type ProductionExecutionService struct {
	DB         *gorm.DB
	Authorizer ExecutionAuthorizer
	RetryAuth  ManualRetryAuthorizer
	WriteGuard ProductionWriteAuthorizer
	Factory    ProductionDownstreamFactory
	Now        func() time.Time
}

func (s *ProductionExecutionService) Queue(ctx context.Context, in ExecutionInput, retry bool) (*ExecutionOutput, error) {
	if s == nil || s.DB == nil || s.Factory == nil || s.WriteGuard == nil || in.TenantID <= 0 || in.OperationTaskID == uuid.Nil || in.ActorID == uuid.Nil || strings.TrimSpace(in.RequestID) == "" || strings.TrimSpace(executionIdempotencyKey(in)) == "" {
		return nil, ErrValidation
	}
	if retry {
		if s.RetryAuth == nil || s.RetryAuth.CanRetry(ctx, in.TenantID, in.ActorID, in.OperationTaskID) != nil {
			return nil, ErrPermissionDenied
		}
	} else if s.Authorizer == nil || s.Authorizer.CanExecute(ctx, in.TenantID, in.ActorID, in.OperationTaskID) != nil {
		return nil, ErrPermissionDenied
	}

	var frozen FrozenDouyinDraft
	var draft PlatformDraft
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := lockTask(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		latest, err := findLatestDraftTx(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		if !IsProductionDouyinTask(task, latest) {
			return ErrExecutionModeForbidden
		}
		frozen, err = ParseFrozenDouyinDraft(latest.Payload)
		if err != nil {
			return err
		}
		draft = *latest
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.WriteGuard.EvaluateWrite(ctx, in.TenantID, frozen.ShopID, frozen.ProductID, frozen.SKUCount); err != nil {
		return nil, ErrPermissionDenied
	}

	var out ExecutionOutput
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		task, err := lockTask(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		latest, err := findLatestDraftTx(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		if latest.ID != draft.ID || latest.PayloadHash != draft.PayloadHash {
			return ErrDraftBindingConflict
		}
		if existing, lookupErr := findAttemptByIdempotencyTx(tx, in.TenantID, in.OperationTaskID, executionIdempotencyKey(in)); lookupErr == nil {
			replay, replayErr := replayAttemptTx(tx, existing, latest, OperationTaskEventActorUser)
			if replayErr != nil {
				return replayErr
			}
			out = *replay
			return nil
		} else if !errors.Is(lookupErr, ErrNotFound) {
			return lookupErr
		}
		if retry {
			if task.Status != OperationTaskStatusExecutionFailed {
				return ErrStateConflict
			}
			latestAttempt, err := latestAttemptTx(tx, in.TenantID, in.OperationTaskID)
			if err != nil || latestAttempt.Status != ExecutionAttemptStatusFailed {
				return ErrStateConflict
			}
			latestErr, err := latestExecutionErrorTx(tx, in.TenantID, latestAttempt.ID)
			if err != nil || !latestErr.Retryable {
				return ErrStateConflict
			}
		} else if !NewTaskStateMachine().CanExecute(task.Status) {
			return ErrStateConflict
		}
		approval, err := findLatestApprovedApprovalTx(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		if approval.PlatformDraftID != latest.ID || approval.DraftVersion != latest.DraftVersion || approval.DraftPayloadHash != latest.PayloadHash {
			return ErrDraftBindingConflict
		}
		if hasActiveAttemptTx(tx, in.TenantID, in.OperationTaskID) {
			return ErrExecutionInProgress
		}
		attemptNumber, err := nextAttemptNumberTx(tx, in.TenantID, in.OperationTaskID)
		if err != nil {
			return err
		}
		key := executionIdempotencyKey(in)
		attempt := ExecutionAttempt{
			TenantID: in.TenantID, OperationTaskID: in.OperationTaskID, PlatformDraftID: latest.ID, ApprovalRecordID: approval.ID,
			AttemptNumber: attemptNumber, Status: ExecutionAttemptStatusQueued, AdapterMode: AdapterModeProductionDraft, Platform: PlatformDouyin,
			ApprovedDraftVersion: approval.DraftVersion, ApprovedDraftPayloadHash: approval.DraftPayloadHash,
			ExecutedDraftVersion: latest.DraftVersion, ExecutedDraftPayloadHash: latest.PayloadHash,
			RequestID: strings.TrimSpace(in.RequestID), IdempotencyKey: &key, SafeMetadata: datatypes.JSON([]byte(`{}`)), Revision: 1,
		}
		if err := validateExecutionAttempt(&attempt); err != nil {
			return err
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return stableError(err, ErrDuplicateExecutionIdem)
		}
		downstreamID, err := s.Factory.CreateFrozenDouyinDraftTask(ctx, tx, ProductionDownstreamInput{
			TenantID: in.TenantID, OperationTaskID: in.OperationTaskID, ExecutionAttemptID: attempt.ID, ActorID: in.ActorID,
			RequestID: strings.TrimSpace(in.RequestID), PayloadHash: latest.PayloadHash, FrozenDraft: frozen,
		})
		if err != nil {
			return err
		}
		if downstreamID == uuid.Nil {
			return ErrValidation
		}
		if err := tx.Model(&ExecutionAttempt{}).Where("id = ? AND tenant_id = ?", attempt.ID, in.TenantID).Update("downstream_task_id", downstreamID).Error; err != nil {
			return err
		}
		attempt.DownstreamTaskID = &downstreamID
		now := s.now()
		outbox := ExecutionOutbox{TenantID: in.TenantID, ExecutionAttemptID: attempt.ID, DownstreamTaskID: downstreamID, Status: ExecutionOutboxStatusPending, NextDispatchAt: now}
		if err := tx.Create(&outbox).Error; err != nil {
			return err
		}
		before := task.Status
		if err := updateTaskStatusRevisionTx(tx, task, OperationTaskStatusExecutionQueued, &in.ActorID); err != nil {
			return err
		}
		if err := appendAuditEventTx(tx, OperationTaskEvent{
			TenantID: in.TenantID, OperationTaskID: in.OperationTaskID, EventType: OperationTaskEventTypeExecutionQueued,
			ActorType: OperationTaskEventActorUser, ActorID: &in.ActorID, BeforeState: before, AfterState: OperationTaskStatusExecutionQueued,
			PlatformDraftID: &latest.ID, DraftVersion: latest.DraftVersion, RequestID: strings.TrimSpace(in.RequestID), Reason: "approved platform draft queued",
			Metadata: safeMetadataJSON(map[string]any{"attemptId": attempt.ID.String(), "downstreamTaskId": downstreamID.String()}),
		}); err != nil {
			return err
		}
		out = ExecutionOutput{Status: ExecutionIdempotencyStatusInProgress, Attempt: attempt}
		return nil
	})
	return &out, err
}

func (s *ProductionExecutionService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

type ProductionResultService struct {
	DB  *gorm.DB
	Now func() time.Time
}

func (s *ProductionResultService) MarkRunning(ctx context.Context, attemptID, downstreamTaskID uuid.UUID) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		attempt, task, err := s.lockBoundAttempt(tx, attemptID, downstreamTaskID)
		if err != nil {
			return err
		}
		if attempt.Status == ExecutionAttemptStatusRunning {
			return nil
		}
		if attempt.Status != ExecutionAttemptStatusQueued || task.Status != OperationTaskStatusExecutionQueued {
			return ErrStateConflict
		}
		now := s.now()
		if err := tx.Model(&ExecutionAttempt{}).Where("id = ? AND status = ?", attempt.ID, ExecutionAttemptStatusQueued).Updates(map[string]any{"status": ExecutionAttemptStatusRunning, "started_at": now, "revision": gorm.Expr("revision + 1"), "updated_at": now}).Error; err != nil {
			return err
		}
		if err := updateTaskStatusRevisionTx(tx, task, OperationTaskStatusExecuting, nil); err != nil {
			return err
		}
		return appendAuditEventTx(tx, OperationTaskEvent{TenantID: attempt.TenantID, OperationTaskID: attempt.OperationTaskID, EventType: OperationTaskEventTypeExecutionStarted, ActorType: OperationTaskEventActorSystem, BeforeState: OperationTaskStatusExecutionQueued, AfterState: OperationTaskStatusExecuting, PlatformDraftID: &attempt.PlatformDraftID, DraftVersion: attempt.ExecutedDraftVersion, RequestID: attempt.RequestID, Reason: "platform worker claimed task", Metadata: safeMetadataJSON(map[string]any{"downstreamTaskId": downstreamTaskID.String()})})
	})
}

func (s *ProductionResultService) MarkSucceeded(ctx context.Context, attemptID, downstreamTaskID uuid.UUID, externalReference, requestID string, metadata datatypes.JSON) error {
	return s.finish(ctx, attemptID, downstreamTaskID, true, false, "", "", false, externalReference, requestID, metadata)
}

func (s *ProductionResultService) MarkFailed(ctx context.Context, attemptID, downstreamTaskID uuid.UUID, code, safeMessage string, retryable, resultUnknown bool, metadata datatypes.JSON) error {
	return s.finish(ctx, attemptID, downstreamTaskID, false, resultUnknown, code, safeMessage, retryable, "", "", metadata)
}

func (s *ProductionResultService) finish(ctx context.Context, attemptID, downstreamTaskID uuid.UUID, success, resultUnknown bool, code, safeMessage string, retryable bool, externalReference, requestID string, metadata datatypes.JSON) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		attempt, task, err := s.lockBoundAttempt(tx, attemptID, downstreamTaskID)
		if err != nil {
			return err
		}
		if success && attempt.Status == ExecutionAttemptStatusSucceeded {
			return nil
		}
		if !success && (attempt.Status == ExecutionAttemptStatusFailed || attempt.Status == ExecutionAttemptStatusResultUnknown) {
			return nil
		}
		recoveredUnknownResult := success && attempt.Status == ExecutionAttemptStatusResultUnknown && task.Status == OperationTaskStatusResultUnknown
		if attempt.Status != ExecutionAttemptStatusRunning && attempt.Status != ExecutionAttemptStatusQueued && !recoveredUnknownResult {
			return ErrStateConflict
		}
		now := s.now()
		attemptStatus, taskStatus, eventType, resultType := ExecutionAttemptStatusSucceeded, OperationTaskStatusDraftWritten, OperationTaskEventTypeDraftWritten, "platform_draft"
		if !success {
			attemptStatus, taskStatus, eventType, resultType = ExecutionAttemptStatusFailed, OperationTaskStatusExecutionFailed, OperationTaskEventTypeExecutionFailed, ""
			if resultUnknown {
				attemptStatus, taskStatus, eventType, resultType, retryable = ExecutionAttemptStatusResultUnknown, OperationTaskStatusResultUnknown, OperationTaskEventTypeResultUnknown, "result_unknown", false
			}
		}
		updates := map[string]any{"status": attemptStatus, "result_type": resultType, "external_reference": strings.TrimSpace(externalReference), "safe_metadata": redactSafeJSON(metadata), "finished_at": now, "revision": gorm.Expr("revision + 1"), "updated_at": now}
		allowedAttemptStatuses := []string{ExecutionAttemptStatusQueued, ExecutionAttemptStatusRunning}
		if recoveredUnknownResult {
			allowedAttemptStatuses = append(allowedAttemptStatuses, ExecutionAttemptStatusResultUnknown)
		}
		if err := tx.Model(&ExecutionAttempt{}).Where("id = ? AND status IN ?", attempt.ID, allowedAttemptStatuses).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&ExecutionOutbox{}).Where("execution_attempt_id = ?", attempt.ID).Updates(map[string]any{"status": ExecutionOutboxStatusDelivered, "delivered_at": now, "last_error_code": "", "updated_at": now}).Error; err != nil {
			return err
		}
		before := task.Status
		if err := updateTaskStatusRevisionTx(tx, task, taskStatus, nil); err != nil {
			return err
		}
		if success {
			if err := tx.Model(&PlatformDraft{}).Where("id = ? AND tenant_id = ?", attempt.PlatformDraftID, attempt.TenantID).Update("status", PlatformDraftStatusWritten).Error; err != nil {
				return err
			}
		} else {
			seq, err := nextExecutionErrorSequenceTx(tx, attempt.TenantID, attempt.ID)
			if err != nil {
				return err
			}
			failure := sanitizeFailure(ExecutionFailure{Category: ExecutionErrorCategoryProviderRejected, Code: code, SafeMessage: safeMessage, Retryable: retryable, ResultCertainty: map[bool]string{true: "unknown", false: "known_failed"}[resultUnknown], Details: metadata})
			if resultUnknown {
				failure.Category = ExecutionErrorCategoryProviderTimeout
			}
			record := ExecutionError{TenantID: attempt.TenantID, ExecutionAttemptID: attempt.ID, Sequence: seq, Category: failure.Category, Code: failure.Code, SafeMessage: failure.SafeMessage, Retryable: failure.Retryable, Details: failure.Details, OccurredAt: now}
			if err := validateExecutionError(&record); err != nil {
				return err
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		reason := "platform draft created"
		if !success {
			reason = strings.TrimSpace(safeMessage)
			if reason == "" {
				reason = "platform draft execution failed"
			}
		}
		return appendAuditEventTx(tx, OperationTaskEvent{TenantID: attempt.TenantID, OperationTaskID: attempt.OperationTaskID, EventType: eventType, ActorType: OperationTaskEventActorSystem, BeforeState: before, AfterState: taskStatus, PlatformDraftID: &attempt.PlatformDraftID, DraftVersion: attempt.ExecutedDraftVersion, RequestID: firstNonEmptyString(requestID, attempt.RequestID), Reason: reason, Metadata: redactSafeJSON(metadata)})
	})
}

func (s *ProductionResultService) lockBoundAttempt(tx *gorm.DB, attemptID, downstreamTaskID uuid.UUID) (*ExecutionAttempt, *OperationTask, error) {
	if s == nil || s.DB == nil || attemptID == uuid.Nil || downstreamTaskID == uuid.Nil {
		return nil, nil, ErrValidation
	}
	var attempt ExecutionAttempt
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND downstream_task_id = ?", attemptID, downstreamTaskID).First(&attempt).Error; err != nil {
		return nil, nil, stableError(err, ErrNotFound)
	}
	task, err := lockTask(tx, attempt.TenantID, attempt.OperationTaskID)
	return &attempt, task, err
}

func (s *ProductionResultService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type OutboxDelivery interface {
	EnqueueProductionTask(ctx context.Context, taskID uuid.UUID) error
}

type OutboxDispatcher struct {
	DB       *gorm.DB
	Delivery OutboxDelivery
	Now      func() time.Time
}

func (d *OutboxDispatcher) DispatchPending(ctx context.Context, limit int) (int, error) {
	if d == nil || d.DB == nil || d.Delivery == nil {
		return 0, fmt.Errorf("operationtask outbox: dependency unavailable")
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	now := d.now()
	var rows []ExecutionOutbox
	if err := d.DB.WithContext(ctx).Where("status = ? AND next_dispatch_at <= ?", ExecutionOutboxStatusPending, now).Order("next_dispatch_at ASC, id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return 0, err
	}
	dispatched := 0
	for _, row := range rows {
		err := d.Delivery.EnqueueProductionTask(ctx, row.DownstreamTaskID)
		next := now.Add(time.Duration(minInt(row.DispatchAttempts+1, 12)) * 5 * time.Second)
		code := ""
		if err != nil {
			code = "queue_unavailable"
		} else {
			dispatched++
		}
		if updateErr := d.DB.WithContext(ctx).Model(&ExecutionOutbox{}).Where("id = ? AND status = ?", row.ID, ExecutionOutboxStatusPending).Updates(map[string]any{"dispatch_attempts": gorm.Expr("dispatch_attempts + 1"), "next_dispatch_at": next, "last_error_code": code, "updated_at": now}).Error; updateErr != nil {
			return dispatched, fmt.Errorf("update operation task outbox %s: %w", row.ID, updateErr)
		}
	}
	return dispatched, nil
}

func (d *OutboxDispatcher) now() time.Time {
	if d != nil && d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
