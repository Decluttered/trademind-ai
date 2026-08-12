package customerchat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/trademind-ai/trademind/backend/internal/modules/worker"
	"github.com/trademind-ai/trademind/backend/internal/pkg/tasktenant"
)

// StartAutoReplyWorker consumes isolated AI auto-reply jobs. Failed jobs remain visible and are never auto-retried.
func StartAutoReplyWorker(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger, svc *Service, concurrency int) {
	if svc == nil || svc.Redis == nil || svc.Redis.Client == nil {
		return
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 8 {
		concurrency = 8
	}
	svc.startAutoReplyRecovery(ctx, wg, log)
	for slot := 1; slot <= concurrency; slot++ {
		wg.Add(1)
		go func(workerSlot int) {
			defer wg.Done()
			markAutoReplyWorkerStarted()
			defer markAutoReplyWorkerStopped()
			workerID := worker.GenerateWorkerID("customer_auto_reply")
			for {
				payload, err := svc.Redis.BRPopLPush(ctx, svc.normalizedAutoReplyQueueName(), svc.autoReplyProcessingQueueName(), 5*time.Second).Result()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}
				msg, shopID, messageID, decodeErr := decodeAutoReplyQueueMessage(payload)
				if decodeErr != nil {
					if log != nil {
						log.Warn("customer_auto_reply_bad_message", "worker", workerSlot)
					}
					_ = svc.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken)
					continue
				}
				jobCtx, cancel := context.WithTimeout(ctx, 150*time.Second)
				workerCtx, _, tenantErr := tasktenant.BeginWorkerWithLegacyZero(jobCtx, svc.DB, 0, shopID, "customer_auto_reply", svc.AllowLegacyTenantZero)
				if tenantErr != nil {
					cancel()
					if log != nil {
						log.Warn("customer_auto_reply_tenant_missing", "worker", workerSlot, "messageId", messageID.String(), "error", tasktenant.WrapError(tenantErr))
					}
					_ = svc.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken)
					continue
				}
				_, runErr := svc.ProcessQueuedAutoReply(workerCtx, shopID, messageID, workerID)
				cancel()
				if ackErr := svc.acknowledgeAutoReplyPayload(ctx, payload, messageID, msg.DispatchToken); ackErr != nil && log != nil {
					log.Warn("customer_auto_reply_ack_failed", "worker", workerSlot, "messageId", messageID.String(), "error", ackErr)
				}
				if runErr != nil && log != nil {
					log.Warn("customer_auto_reply_failed", "worker", workerSlot, "messageId", messageID.String(), "error", runErr)
				}
			}
		}(slot)
	}
}
