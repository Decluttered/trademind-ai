package customerchat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/trademind-ai/trademind/backend/internal/modules/shop"
)

const (
	autoReplyDispatchTTL      = 4 * time.Minute
	autoReplyRecoveryInterval = 30 * time.Second
	autoReplyRecoveryBatch    = 200
)

func (s *Service) autoReplyProcessingQueueName() string {
	return s.normalizedAutoReplyQueueName() + ":processing"
}

func autoReplyDispatchKey(messageID uuid.UUID) string {
	return "customer:auto:reply:dispatch:" + messageID.String()
}

func (s *Service) enqueueAutoReplyWakeup(ctx context.Context, shopID, messageID uuid.UUID) error {
	if s == nil || s.Redis == nil || s.Redis.Client == nil {
		return fmt.Errorf("auto reply queue unavailable")
	}
	key := autoReplyDispatchKey(messageID)
	token := uuid.NewString()
	reserved, err := s.Redis.SetNX(ctx, key, token, autoReplyDispatchTTL).Result()
	if err != nil {
		return fmt.Errorf("reserve auto reply dispatch: %w", err)
	}
	if !reserved {
		return nil
	}
	payload, err := json.Marshal(autoReplyQueueMessage{ShopID: shopID.String(), MessageID: messageID.String(), DispatchToken: token})
	if err != nil {
		_ = releaseRedisOwnerToken(context.Background(), s.Redis.Client, key, token)
		return fmt.Errorf("encode auto reply queue message: %w", err)
	}
	if err := s.Redis.LPush(ctx, s.normalizedAutoReplyQueueName(), string(payload)).Err(); err != nil {
		_ = releaseRedisOwnerToken(context.Background(), s.Redis.Client, key, token)
		return fmt.Errorf("enqueue auto reply: %w", err)
	}
	return nil
}

func releaseRedisOwnerToken(ctx context.Context, client redis.Cmdable, key, token string) error {
	if client == nil {
		return nil
	}
	const script = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0`
	return client.Eval(ctx, script, []string{key}, token).Err()
}

func (s *Service) acknowledgeAutoReplyPayload(ctx context.Context, payload string, messageID uuid.UUID, dispatchToken string) error {
	if s == nil || s.Redis == nil || s.Redis.Client == nil {
		return nil
	}
	pipe := s.Redis.TxPipeline()
	pipe.LRem(ctx, s.autoReplyProcessingQueueName(), 1, payload)
	_, err := pipe.Exec(ctx)
	if err != nil || messageID == uuid.Nil || strings.TrimSpace(dispatchToken) == "" {
		return err
	}
	return releaseRedisOwnerToken(ctx, s.Redis.Client, autoReplyDispatchKey(messageID), dispatchToken)
}

func decodeAutoReplyQueueMessage(payload string) (autoReplyQueueMessage, uuid.UUID, uuid.UUID, error) {
	var msg autoReplyQueueMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return msg, uuid.Nil, uuid.Nil, err
	}
	shopID, shopErr := uuid.Parse(strings.TrimSpace(msg.ShopID))
	messageID, messageErr := uuid.Parse(strings.TrimSpace(msg.MessageID))
	if shopErr != nil || messageErr != nil {
		return msg, uuid.Nil, uuid.Nil, fmt.Errorf("invalid shop or message id")
	}
	return msg, shopID, messageID, nil
}

func (s *Service) recoverAutoReplyQueue(ctx context.Context, log *slog.Logger) error {
	if s == nil || s.DB == nil || s.Redis == nil || s.Redis.Client == nil {
		return nil
	}
	payloads, err := s.Redis.LRange(ctx, s.autoReplyProcessingQueueName(), 0, autoReplyRecoveryBatch-1).Result()
	if err != nil {
		return err
	}
	for _, payload := range payloads {
		msg, shopID, messageID, decodeErr := decodeAutoReplyQueueMessage(payload)
		if decodeErr != nil {
			_ = s.acknowledgeAutoReplyPayload(ctx, payload, uuid.Nil, "")
			continue
		}
		var run CustomerAutoReplyRun
		findErr := s.DB.WithContext(ctx).Where("message_id = ? AND shop_id = ?", messageID, shopID).First(&run).Error
		if findErr != nil {
			_ = s.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken)
			continue
		}
		now := time.Now().UTC()
		if (run.Status == AutoReplyRunGenerating || run.Status == AutoReplyRunSending) && run.LockedUntil != nil && run.LockedUntil.Before(now) {
			if recoverErr := s.recoverExpiredAutoReplyRun(ctx, run.ID); recoverErr != nil {
				if log != nil {
					log.Warn("customer_auto_reply_recover_run_failed", "runId", run.ID.String(), "error", recoverErr)
				}
				continue
			}
			if err := s.DB.WithContext(ctx).First(&run, "id = ?", run.ID).Error; err != nil {
				continue
			}
		}
		switch run.Status {
		case AutoReplyRunPending:
			if err := s.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken); err == nil {
				_ = s.enqueueAutoReplyWakeup(ctx, shopID, messageID)
			}
		case AutoReplyRunGenerating, AutoReplyRunSending:
			// An unexpired lease owns this payload.
		default:
			_ = s.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken)
		}
	}

	var expired []CustomerAutoReplyRun
	now := time.Now().UTC()
	if err := s.DB.WithContext(ctx).
		Where("status IN ? AND locked_until IS NOT NULL AND locked_until < ?", []string{AutoReplyRunGenerating, AutoReplyRunSending}, now).
		Order("locked_until ASC").Limit(autoReplyRecoveryBatch).Find(&expired).Error; err != nil {
		return err
	}
	for i := range expired {
		wasGenerating := expired[i].Status == AutoReplyRunGenerating
		if err := s.recoverExpiredAutoReplyRun(ctx, expired[i].ID); err != nil {
			if log != nil {
				log.Warn("customer_auto_reply_expired_run_failed", "runId", expired[i].ID.String(), "error", err)
			}
			continue
		}
		if wasGenerating {
			_ = s.enqueueAutoReplyWakeup(ctx, expired[i].ShopID, expired[i].MessageID)
		}
	}

	var pending []CustomerAutoReplyRun
	if err := s.DB.WithContext(ctx).Where("status = ?", AutoReplyRunPending).
		Order("created_at ASC").Limit(autoReplyRecoveryBatch).Find(&pending).Error; err != nil {
		return err
	}
	for i := range pending {
		var messageCount, shopCount int64
		if err := s.DB.WithContext(ctx).Model(&CustomerMessage{}).Where("id = ?", pending[i].MessageID).Count(&messageCount).Error; err != nil {
			continue
		}
		if err := s.DB.WithContext(ctx).Model(&shop.Shop{}).Where("id = ?", pending[i].ShopID).Count(&shopCount).Error; err != nil {
			continue
		}
		if messageCount == 0 || shopCount == 0 {
			_ = s.finishPendingAutoReplyRun(ctx, pending[i].ID, AutoReplyRunFailed, "source_record_missing", "referenced shop or customer message no longer exists")
			continue
		}
		if err := s.enqueueAutoReplyWakeup(ctx, pending[i].ShopID, pending[i].MessageID); err != nil && log != nil {
			log.Warn("customer_auto_reply_pending_dispatch_failed", "runId", pending[i].ID.String(), "error", err)
		}
	}
	return nil
}

func (s *Service) startAutoReplyRecovery(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger) {
	if wg == nil {
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(autoReplyRecoveryInterval)
		defer ticker.Stop()
		for {
			if err := s.recoverAutoReplyQueue(ctx, log); err != nil && log != nil && ctx.Err() == nil {
				log.Warn("customer_auto_reply_recovery_failed", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
