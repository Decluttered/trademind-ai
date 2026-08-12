package collect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	collectRecoveryLockTTL  = 30 * time.Second
	collectPendingCutoff    = time.Minute
	collectRecoveryPageSize = 200
)

func (s *Service) failTaskMissingTenant(ctx context.Context, task *CollectTask) error {
	if s == nil || s.DB == nil || task == nil {
		return fmt.Errorf("collect: missing tenant failure unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	message := "任务缺少租户上下文，无法处理"
	fromStatus := task.Status
	if fromStatus != StatusPending && fromStatus != StatusRetrying {
		return nil
	}
	finishedAt := time.Now().UTC()
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&CollectTask{}).
			Where("id = ? AND status = ?", task.ID, fromStatus).
			Updates(map[string]any{
				"status":            StatusFailed,
				"error_message":     message,
				"finished_at":       &finishedAt,
				"next_retry_at":     nil,
				"retry_enqueued_at": nil,
				"locked_by":         nil,
				"locked_until":      nil,
				"updated_at":        finishedAt,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		task.Status = StatusFailed
		task.ErrorMessage = message
		task.FinishedAt = &finishedAt
		if err := s.recordCollectTaskEvent(tx, ctx, task, TaskEventInput{
			EventType:    EventTaskFailed,
			FromStatus:   fromStatus,
			ToStatus:     StatusFailed,
			Message:      "collect task rejected before claim",
			ErrorMessage: message,
			RetryCount:   task.RetryCount,
			MaxRetries:   s.effectiveMaxRetries(task),
		}); err != nil {
			return err
		}
		if task.BatchID != nil {
			return s.reconcileCollectBatchTx(ctx, tx, *task.BatchID)
		}
		return nil
	})
}

func (s *Service) recoverCollectQueue(ctx context.Context, queueName string, log *slog.Logger) error {
	if s == nil || s.DB == nil || s.Redis == nil || s.Redis.Client == nil {
		return fmt.Errorf("collect: recovery dependencies unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	lockKey := collectQueueName(queueName) + ":recovery-lock"
	token := uuid.NewString()
	locked, err := s.Redis.SetNX(ctx, lockKey, token, collectRecoveryLockTTL).Result()
	if err != nil || !locked {
		return err
	}
	defer s.releaseCollectRecoveryLock(lockKey, token)
	if err := s.recoverProcessingTasks(ctx, queueName, log); err != nil {
		return err
	}
	return s.recoverStalePendingTasks(ctx, queueName, log)
}

func (s *Service) releaseCollectRecoveryLock(lockKey, token string) {
	if s == nil || s.Redis == nil || s.Redis.Client == nil {
		return
	}
	const unlockScript = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_ = s.Redis.Eval(context.Background(), unlockScript, []string{lockKey}, token).Err()
}

func (s *Service) recoverProcessingTasks(ctx context.Context, queueName string, log *slog.Logger) error {
	processing := collectProcessingQueueName(queueName)
	payloads, err := s.Redis.LRange(ctx, processing, 0, -1).Result()
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		var msg QueueMessage
		if err := json.Unmarshal([]byte(payload), &msg); err != nil {
			if err := s.ackReservedTask(ctx, queueName, payload); err != nil {
				return err
			}
			continue
		}
		taskID, err := uuid.Parse(strings.TrimSpace(msg.TaskID))
		if err != nil {
			if err := s.ackReservedTask(ctx, queueName, payload); err != nil {
				return err
			}
			continue
		}
		var task CollectTask
		if err := s.DB.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := s.ackReservedTask(ctx, queueName, payload); err != nil {
					return err
				}
				continue
			}
			return err
		}
		switch {
		case task.Status == StatusPending || (task.Status == StatusRetrying && task.NextRetryAt == nil):
			if err := s.requeueReservedTask(ctx, queueName, payload); err != nil {
				return err
			}
		case task.Status != StatusRunning:
			if err := s.ackReservedTask(ctx, queueName, payload); err != nil {
				return err
			}
		default:
			if log != nil {
				log.Info("collect_processing_task_still_running", "taskId", task.ID.String())
			}
		}
	}
	return nil
}

func (s *Service) stalePendingTasksPage(ctx context.Context, cutoff, cursorTime time.Time, cursorID uuid.UUID) ([]CollectTask, error) {
	var rows []CollectTask
	query := s.DB.WithContext(ctx).
		Where("updated_at <= ? AND ((status = ?) OR (status = ? AND next_retry_at IS NULL))", cutoff, StatusPending, StatusRetrying).
		Where("locked_by IS NULL OR locked_until < ?", time.Now().UTC())
	if !cursorTime.IsZero() {
		query = query.Where("created_at > ? OR (created_at = ? AND id > ?)", cursorTime, cursorTime, cursorID)
	}
	if err := query.Order("created_at ASC, id ASC").Limit(collectRecoveryPageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) queuedCollectTaskIDs(ctx context.Context, queueName string) (map[uuid.UUID]struct{}, error) {
	payloads, err := s.Redis.LRange(ctx, collectQueueName(queueName), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	processing, err := s.Redis.LRange(ctx, collectProcessingQueueName(queueName), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	payloads = append(payloads, processing...)
	queued := make(map[uuid.UUID]struct{}, len(payloads))
	for _, payload := range payloads {
		var msg QueueMessage
		if err := json.Unmarshal([]byte(payload), &msg); err != nil {
			continue
		}
		id, err := uuid.Parse(strings.TrimSpace(msg.TaskID))
		if err == nil {
			queued[id] = struct{}{}
		}
	}
	return queued, nil
}

func (s *Service) recoverStalePendingTasks(ctx context.Context, queueName string, log *slog.Logger) error {
	queued, err := s.queuedCollectTaskIDs(ctx, queueName)
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-collectPendingCutoff)
	var cursorTime time.Time
	var cursorID uuid.UUID
	for {
		rows, err := s.stalePendingTasksPage(ctx, cutoff, cursorTime, cursorID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			task := &rows[i]
			if !s.acceptsTenantID(task.TenantID) {
				if err := s.failTaskMissingTenant(ctx, task); err != nil {
					return err
				}
				continue
			}
			if _, exists := queued[task.ID]; exists {
				continue
			}
			payload, err := marshalQueueMessage(task.ID, task.Source, task.SourceURL, task.CreatedBy, "startup-recovery")
			if err != nil {
				return err
			}
			if err := s.Redis.LPush(ctx, collectQueueName(queueName), payload).Err(); err != nil {
				return err
			}
			if log != nil {
				log.Info("collect_pending_task_requeued", "taskId", task.ID.String())
			}
		}
		last := rows[len(rows)-1]
		cursorTime = last.CreatedAt
		cursorID = last.ID
		if len(rows) < collectRecoveryPageSize {
			return nil
		}
	}
}
