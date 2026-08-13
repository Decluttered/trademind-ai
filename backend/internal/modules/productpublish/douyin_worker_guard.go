package productpublish

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	douyinmetrics "github.com/trademind-ai/trademind/backend/internal/metrics/douyin"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationtask"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
)

func (s *Service) guardDouyinWorker(ctx context.Context, taskID uuid.UUID, shopID uuid.UUID, feature string, isScheduled bool, createdBy *uuid.UUID) error {
	if ge := platformdouyin.GuardWorkerWithShop(ctx, shopID.String(), feature, true, isScheduled); ge != nil {
		douyinmetrics.RecordRuntimeBlockedTask()
		return s.blockDouyinTask(ctx, taskID, ge, createdBy)
	}
	return nil
}

func (s *Service) blockDouyinTask(ctx context.Context, taskID uuid.UUID, ge *platformdouyin.Error, createdBy *uuid.UUID) error {
	if s == nil || s.DB == nil || ge == nil {
		return ge
	}
	fin := time.Now().UTC()
	out := platformdouyin.MarshalRecoveryOutput(nil, platformdouyin.TaskRecoveryMeta{
		RecoveryStatus: platformdouyin.RecoverySkipped,
		LastErrorCode:  ge.Code,
		UserMessage:    ge.Message,
		TechnicalCode:  ge.Code,
	})
	_ = s.DB.WithContext(ctx).Model(&ProductPublishTask{}).Where("id = ?", taskID).
		Updates(map[string]any{
			"status":         TaskCancelled,
			"publish_status": StatusPubFailed,
			"error_code":     ge.Code,
			"error_message":  ge.Message,
			"retryable":      false,
			"finished_at":    &fin,
			"output":         datatypes.JSON(out),
			"locked_by":      nil,
			"locked_until":   nil,
			"updated_at":     fin,
		}).Error
	return ge
}

func (s *Service) markDouyinStale(ctx context.Context, taskID uuid.UUID, code, recoveryStatus string, createdBy *uuid.UUID) {
	if s == nil || s.DB == nil {
		return
	}
	douyinmetrics.RecordStaleTask()
	fin := time.Now().UTC()
	meta := platformdouyin.TaskRecoveryMeta{
		RecoveryStatus: recoveryStatus,
		LastErrorCode:  code,
		UserMessage:    platformdouyin.UserMessageForRecovery(recoveryStatus),
		TechnicalCode:  code,
	}
	out := platformdouyin.MarshalRecoveryOutput(nil, meta)
	_ = s.DB.WithContext(ctx).Model(&ProductPublishTask{}).Where("id = ?", taskID).
		Updates(map[string]any{
			"status":         TaskFailed,
			"publish_status": StatusPubFailed,
			"error_code":     code,
			"error_message":  meta.UserMessage,
			"retryable":      false,
			"finished_at":    &fin,
			"output":         datatypes.JSON(out),
			"locked_by":      nil,
			"locked_until":   nil,
			"updated_at":     fin,
		}).Error
}

func (s *Service) touchDouyinTaskProgress(ctx context.Context, taskID uuid.UUID, patch map[string]any) {
	if s == nil || s.DB == nil {
		return
	}
	patch["updated_at"] = time.Now().UTC()
	_ = s.DB.WithContext(ctx).Model(&ProductPublishTask{}).Where("id = ?", taskID).Updates(patch).Error
}

func parseTaskOutputMap(raw datatypes.JSON) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func mergeTaskOutput(existing datatypes.JSON, patch map[string]any) datatypes.JSON {
	base := parseTaskOutputMap(existing)
	for k, v := range patch {
		base[k] = v
	}
	b, _ := json.Marshal(base)
	return datatypes.JSON(b)
}

// RecoverDouyinDraftStale attempts product.detail recovery for result_unknown tasks.
func (s *Service) RecoverDouyinDraftStale(ctx context.Context, taskID uuid.UUID) error {
	if s == nil || s.DB == nil || taskID == uuid.Nil {
		return ErrDouyinRecoveryNotAllowed
	}
	var task ProductPublishTask
	if err := s.DB.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		return err
	}
	if task.Platform != "douyin_shop" || task.Status != TaskFailed || task.ExecutionAttemptID == nil || *task.ExecutionAttemptID == uuid.Nil || task.OperationTaskID == nil || *task.OperationTaskID == uuid.Nil || !isDouyinResultUnknownTask(&task) {
		return ErrDouyinRecoveryNotAllowed
	}
	var attempt operationtask.ExecutionAttempt
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ? AND operation_task_id = ? AND downstream_task_id = ? AND status = ?", *task.ExecutionAttemptID, task.TenantID, *task.OperationTaskID, task.ID, operationtask.ExecutionAttemptStatusResultUnknown).First(&attempt).Error; err != nil {
		return ErrDouyinRecoveryNotAllowed
	}
	var operationTask operationtask.OperationTask
	if err := s.DB.WithContext(ctx).Where("id = ? AND tenant_id = ? AND status = ?", *task.OperationTaskID, task.TenantID, operationtask.OperationTaskStatusResultUnknown).First(&operationTask).Error; err != nil {
		return ErrDouyinRecoveryNotAllowed
	}
	if s.Shops == nil || s.WriteControl == nil || s.OperationResults == nil {
		return ErrDouyinRecoveryNotAllowed
	}
	snap, buildRes, err := s.validateProductionDouyinTask(ctx, &task)
	if err != nil {
		return err
	}
	if guardErr := platformdouyin.GuardWorkerWithShop(ctx, task.ShopID.String(), platformdouyin.FeatureProductDraft, false, false); guardErr != nil {
		return guardErr
	}
	if err := s.WriteControl.EvaluateWrite(ctx, task.TenantID, task.ShopID, task.ProductID, len(buildRes.APIReq.SpecPricesV2)); err != nil {
		return err
	}
	client, _, err := s.Shops.DouyinClientForShopContext(ctx, task.ShopID, task.CreatedBy)
	if err != nil {
		return err
	}
	res, recovered, recErr := tryRecoverDouyinDraftFromPlatform(ctx, client, task.ShopID.String(), buildRes.APIReq.OuterProductID)
	if recErr != nil {
		return recErr
	}
	if !recovered || res == nil {
		s.markDouyinStale(ctx, taskID, platformdouyin.CodeDouyinTaskRecoveryRequired, platformdouyin.RecoveryRequired, task.CreatedBy)
		douyinmetrics.RecordRecoveryFailed()
		return nil
	}
	douyinmetrics.RecordRecoverySuccess()
	return s.completeDouyinDraftSuccess(ctx, &task, taskID, "", nil, snap, buildRes, res)
}

func isDouyinResultUnknownTask(task *ProductPublishTask) bool {
	if task == nil {
		return false
	}
	if task.ErrorCode == platformdouyin.CodeDouyinUnknownResult || task.ErrorCode == platformdouyin.CodeDouyinRequestTimeout {
		return true
	}
	var output struct {
		RecoveryStatus string `json:"recoveryStatus"`
	}
	return len(task.Output) > 0 && json.Unmarshal(task.Output, &output) == nil && strings.EqualFold(strings.TrimSpace(output.RecoveryStatus), platformdouyin.RecoveryResultUnknown)
}
