package customerchat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/tasklease"
	"gorm.io/gorm"
)

const autoReplyLeaseTTL = 180 * time.Second

type autoReplyLease struct {
	WorkerID string
	Claim    tasklease.ClaimResult
}

func (s *Service) claimAutoReplyRun(ctx context.Context, runID uuid.UUID, workerID string) (*CustomerAutoReplyRun, *autoReplyLease, bool, error) {
	claim, ok, err := tasklease.TryClaim(ctx, s.DB, CustomerAutoReplyRun{}.TableName(), AutoReplyRunPending, AutoReplyRunGenerating, runID, workerID, autoReplyLeaseTTL)
	if err != nil || !ok {
		return nil, nil, ok, err
	}
	var run CustomerAutoReplyRun
	if err := s.DB.WithContext(ctx).First(&run, "id = ?", runID).Error; err != nil {
		return nil, nil, false, err
	}
	return &run, &autoReplyLease{WorkerID: workerID, Claim: claim}, true, nil
}

func (s *Service) startAutoReplyLeaseRenewal(ctx context.Context, runID uuid.UUID, lease *autoReplyLease) func() {
	if lease == nil {
		return func() {}
	}
	return tasklease.StartRenewal(ctx, s.DB, CustomerAutoReplyRun{}.TableName(), AutoReplyRunGenerating, runID, lease.WorkerID, lease.Claim.ExecutionID, lease.Claim.LeaseVersion, autoReplyLeaseTTL)
}

func (s *Service) transitionAutoReplyToSending(ctx context.Context, runID uuid.UUID, lease *autoReplyLease) error {
	if lease == nil {
		return tasklease.ErrLeaseLost
	}
	now := time.Now().UTC()
	until := now.Add(autoReplyLeaseTTL)
	res := s.DB.WithContext(ctx).Model(&CustomerAutoReplyRun{}).
		Where("id = ? AND status = ? AND locked_by = ? AND execution_id = ? AND lock_version = ? AND locked_until >= ?",
			runID, AutoReplyRunGenerating, lease.WorkerID, lease.Claim.ExecutionID.String(), lease.Claim.LeaseVersion, now).
		Updates(map[string]any{"status": AutoReplyRunSending, "locked_until": &until, "heartbeat_at": &now, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return tasklease.ErrLeaseLost
	}
	return nil
}

func (s *Service) finishClaimedAutoReplyRun(ctx context.Context, runID uuid.UUID, lease *autoReplyLease, status, risk, reason, errMsg string, suggestionID, sentMessageID *uuid.UUID) error {
	if lease == nil {
		return tasklease.ErrLeaseLost
	}
	now := time.Now().UTC()
	updates := autoReplyRunFinishUpdates(now, status, risk, reason, errMsg, suggestionID, sentMessageID)
	updates["locked_by"] = nil
	updates["locked_until"] = nil
	updates["heartbeat_at"] = nil
	res := s.DB.WithContext(ctx).Model(&CustomerAutoReplyRun{}).
		Where("id = ? AND status IN ? AND locked_by = ? AND execution_id = ? AND lock_version = ? AND locked_until >= ?",
			runID, []string{AutoReplyRunGenerating, AutoReplyRunSending}, lease.WorkerID, lease.Claim.ExecutionID.String(), lease.Claim.LeaseVersion, now).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return tasklease.ErrLeaseLost
	}
	return nil
}

func (s *Service) finishPendingAutoReplyRun(ctx context.Context, runID uuid.UUID, status, reason, errMsg string) error {
	now := time.Now().UTC()
	updates := autoReplyRunFinishUpdates(now, status, "", reason, errMsg, nil, nil)
	res := s.DB.WithContext(ctx).Model(&CustomerAutoReplyRun{}).Where("id = ? AND status = ?", runID, AutoReplyRunPending).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("auto reply pending run transition lost")
	}
	return nil
}

func autoReplyRunFinishUpdates(now time.Time, status, risk, reason, errMsg string, suggestionID, sentMessageID *uuid.UUID) map[string]any {
	return map[string]any{
		"status": status, "risk_level": risk, "reason_code": reason,
		"error_message": truncateRunes(errMsg, 1000), "finished_at": &now,
		"suggestion_id": suggestionID, "sent_message_id": sentMessageID, "updated_at": now,
	}
}

func (s *Service) recoverExpiredAutoReplyRun(ctx context.Context, runID uuid.UUID) error {
	var run CustomerAutoReplyRun
	if err := s.DB.WithContext(ctx).First(&run, "id = ?", runID).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	if run.LockedUntil == nil || !run.LockedUntil.Before(now) {
		return nil
	}
	switch run.Status {
	case AutoReplyRunGenerating:
		res := s.DB.WithContext(ctx).Model(&CustomerAutoReplyRun{}).
			Where("id = ? AND status = ? AND locked_until < ?", run.ID, AutoReplyRunGenerating, now).
			Updates(map[string]any{"status": AutoReplyRunPending, "locked_by": nil, "locked_until": nil, "heartbeat_at": nil, "execution_id": nil, "updated_at": now})
		return res.Error
	case AutoReplyRunSending:
		return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			updates := autoReplyRunFinishUpdates(now, AutoReplyRunHumanRequired, run.RiskLevel, "platform_send_result_unknown", "worker lease expired during platform send; manual review required", run.SuggestionID, nil)
			updates["locked_by"] = nil
			updates["locked_until"] = nil
			updates["heartbeat_at"] = nil
			res := tx.Model(&CustomerAutoReplyRun{}).
				Where("id = ? AND status = ? AND locked_until < ?", run.ID, AutoReplyRunSending, now).
				Updates(updates)
			if res.Error != nil || res.RowsAffected == 0 {
				return res.Error
			}
			var conv CustomerConversation
			if err := tx.Select("id, platform, shop_id").First(&conv, "id = ?", run.ConversationID).Error; err != nil {
				return fmt.Errorf("record failure context: %w", err)
			}
			txService := *s
			txService.DB = tx
			if err := txService.recordFailure(ctx, CustomerFailureEvent{
				ConversationID: run.ConversationID,
				Platform:       conv.Platform,
				ShopID:         conv.ShopID,
				Category:       FailureCategoryReplySendFailed,
				ErrorMessage:   "platform send result unknown; manual review required",
				Status:         FailureEventStatusOpen,
			}); err != nil {
				return fmt.Errorf("record failure: %w", err)
			}
			return nil
		})
	default:
		return nil
	}
}
