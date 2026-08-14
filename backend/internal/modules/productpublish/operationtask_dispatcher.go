package productpublish

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	platformdouyin "github.com/trademind-ai/trademind/backend/internal/providers/platform/douyinshop"
	"gorm.io/datatypes"
)

func StartProductionOutboxDispatcher(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger, svc *Service, interval time.Duration) {
	if svc == nil || svc.ProductionOutbox == nil || !svc.QueueEnabled || svc.Redis == nil || svc.Redis.Client == nil {
		return
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if _, err := svc.ProductionOutbox.DispatchPending(ctx, 50); err != nil && log != nil {
				log.Warn("operation_task_outbox_dispatch_failed", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// ReconcileOperationTaskResult makes result propagation replayable when a
// process exits after the platform task committed but before its aggregate did.
func (s *Service) ReconcileOperationTaskResult(ctx context.Context, taskID uuid.UUID) error {
	if s == nil || s.DB == nil || s.OperationResults == nil || taskID == uuid.Nil {
		return nil
	}
	var task ProductPublishTask
	if err := s.DB.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		return err
	}
	if task.ExecutionAttemptID == nil || *task.ExecutionAttemptID == uuid.Nil {
		return nil
	}
	meta := task.Output
	if len(meta) == 0 {
		meta = datatypes.JSON([]byte(`{}`))
	}
	switch task.Status {
	case TaskSuccess:
		return s.OperationResults.MarkSucceeded(ctx, *task.ExecutionAttemptID, task.ID, task.PlatformProductID, task.RequestID, meta)
	case TaskFailed, TaskCancelled:
		resultUnknown := task.ErrorCode == platformdouyin.CodeDouyinUnknownResult || task.ErrorCode == platformdouyin.CodeDouyinRequestTimeout
		if !resultUnknown && len(task.Output) > 0 {
			var output struct {
				RecoveryStatus string `json:"recoveryStatus"`
			}
			_ = json.Unmarshal(task.Output, &output)
			resultUnknown = strings.EqualFold(output.RecoveryStatus, platformdouyin.RecoveryResultUnknown)
		}
		return s.OperationResults.MarkFailed(ctx, *task.ExecutionAttemptID, task.ID, task.ErrorCode, task.ErrorMessage, task.Retryable && !resultUnknown, resultUnknown, meta)
	default:
		return nil
	}
}
