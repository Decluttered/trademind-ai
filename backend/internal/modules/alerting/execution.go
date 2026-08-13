package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"gorm.io/gorm"
)

const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"

	EvaluationSucceeded = "succeeded"
	EvaluationFailed    = "failed"
	EvaluationWarmingUp = "warming_up"
)

// AlertDelivery stores idempotent alert notification attempts.
type AlertDelivery struct {
	ID             string `gorm:"primaryKey;size:36"`
	AlertID        string `gorm:"size:36;index"`
	RuleID         string `gorm:"size:64;index"`
	Channel        string `gorm:"size:64;index"`
	Status         string `gorm:"size:32;index"`
	Attempt        int
	IdempotencyKey string    `gorm:"size:128;uniqueIndex"`
	SafePayload    string    `gorm:"type:text"`
	LastError      string    `gorm:"type:text"`
	NextRunAt      time.Time `gorm:"index"`
	DeliveredAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (AlertDelivery) TableName() string { return "alert_deliveries" }

// AlertEvaluationRun records periodic rule evaluation evidence.
type AlertEvaluationRun struct {
	ID             string `gorm:"primaryKey;size:36"`
	StartedAt      time.Time
	FinishedAt     *time.Time
	Status         string `gorm:"size:32;index"`
	RulesChecked   int
	RulesSkipped   int
	AlertsFired    int
	AlertsResolved int
	ErrorMessage   string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (AlertEvaluationRun) TableName() string { return "alert_evaluation_runs" }

// EvaluateSnapshot evaluates enabled rules against a bounded metric window.
func (s *Service) EvaluateSnapshot(ctx context.Context, current metrics.Snapshot) (*AlertEvaluationRun, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("alerting unavailable")
	}
	if current.TakenAt.IsZero() {
		current.TakenAt = time.Now().UTC()
	}
	now := current.TakenAt.UTC()
	run := &AlertEvaluationRun{ID: uuid.New().String(), StartedAt: now, Status: EvaluationSucceeded}
	if err := s.DB.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	var rules []AlertRule
	if err := s.DB.WithContext(ctx).Where("enabled = ?", true).Find(&rules).Error; err != nil {
		s.finishEvaluation(ctx, run, EvaluationFailed, err.Error())
		return run, err
	}
	history := s.recordSnapshot(current)
	for _, rule := range rules {
		value, sampleCount, ready, err := evaluateRule(rule, history, current)
		if err != nil {
			s.finishEvaluation(ctx, run, EvaluationFailed, err.Error())
			return run, err
		}
		if !ready {
			run.RulesSkipped++
			continue
		}
		run.RulesChecked++
		firing := compare(value, rule.Condition, rule.Threshold)
		if firing {
			details := fmt.Sprintf("metric=%s value=%.6f threshold=%.6f window=%s samples=%d", rule.Metric, value, rule.Threshold, rule.Window, sampleCount)
			if _, err := s.Fire(ctx, rule.ID, rule.Severity, metricModule(rule.Metric), rule.Name, details); err != nil {
				s.finishEvaluation(ctx, run, EvaluationFailed, err.Error())
				return run, err
			}
			run.AlertsFired++
			continue
		}
		resolved, err := s.resolveActiveForRule(ctx, rule.ID)
		if err != nil {
			s.finishEvaluation(ctx, run, EvaluationFailed, err.Error())
			return run, err
		}
		run.AlertsResolved += resolved
	}
	if run.RulesChecked == 0 && run.RulesSkipped > 0 {
		s.finishEvaluation(ctx, run, EvaluationWarmingUp, "waiting for a complete metric window")
		return run, nil
	}
	s.finishEvaluation(ctx, run, EvaluationSucceeded, "")
	return run, nil
}

func (s *Service) recordSnapshot(current metrics.Snapshot) []metrics.Snapshot {
	s.evaluationMu.Lock()
	defer s.evaluationMu.Unlock()
	cutoff := current.TakenAt.Add(-time.Hour)
	history := make([]metrics.Snapshot, 0, len(s.snapshotHistory)+1)
	for _, snapshot := range s.snapshotHistory {
		if !snapshot.TakenAt.Before(cutoff) && snapshot.TakenAt.Before(current.TakenAt) {
			history = append(history, snapshot)
		}
	}
	history = append(history, current)
	if len(history) > 120 {
		history = history[len(history)-120:]
	}
	s.snapshotHistory = history
	return append([]metrics.Snapshot(nil), history...)
}

func (s *Service) finishEvaluation(ctx context.Context, run *AlertEvaluationRun, status, msg string) {
	if s == nil || s.DB == nil || run == nil {
		return
	}
	now := time.Now().UTC()
	run.FinishedAt = &now
	run.Status = status
	run.ErrorMessage = sanitizeDetails(msg)
	_ = s.DB.WithContext(ctx).Save(run).Error
}

func (s *Service) resolveActiveForRule(ctx context.Context, ruleID string) (int, error) {
	var alerts []AlertEvent
	if err := s.DB.WithContext(ctx).Where("rule_id = ? AND status IN ?", ruleID, []string{StatusFiring, StatusAcknowledged}).Find(&alerts).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, alert := range alerts {
		if err := s.Resolve(ctx, alert.ID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Service) enqueueAndSend(ctx context.Context, ev AlertEvent, n AlertNotification) {
	if s == nil || s.DB == nil {
		return
	}
	for _, ch := range s.Channels {
		if ch == nil {
			continue
		}
		name := strings.TrimSpace(ch.Name())
		if name == "" {
			name = "internal"
		}
		d := AlertDelivery{
			ID:             uuid.New().String(),
			AlertID:        ev.ID,
			RuleID:         ev.RuleID,
			Channel:        name,
			Status:         DeliveryPending,
			Attempt:        0,
			IdempotencyKey: ev.ID + ":" + name + ":" + ev.Status,
			SafePayload:    sanitizeDetails(n.Summary + "\n" + n.SafeDetails),
			NextRunAt:      time.Now().UTC(),
		}
		_ = s.DB.WithContext(ctx).FirstOrCreate(&d, AlertDelivery{IdempotencyKey: d.IdempotencyKey}).Error
		_ = s.DeliverPending(ctx, 20)
	}
}

// DeliverPending sends due delivery rows with idempotent retry state.
func (s *Service) DeliverPending(ctx context.Context, limit int) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("alerting unavailable")
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []AlertDelivery
	if err := s.DB.WithContext(ctx).
		Where("status IN ? AND next_run_at <= ?", []string{DeliveryPending, DeliveryFailed}, time.Now().UTC()).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		ch := s.channelByName(row.Channel)
		if ch == nil {
			row.Status = DeliveryFailed
			row.Attempt++
			row.LastError = "channel unavailable"
			row.NextRunAt = time.Now().UTC().Add(retryBackoff(row.Attempt))
			_ = s.DB.WithContext(ctx).Save(&row).Error
			continue
		}
		n := AlertNotification{AlertID: row.AlertID, RuleID: row.RuleID, Status: StatusFiring, SafeDetails: row.SafePayload}
		if err := ch.Send(ctx, n); err != nil {
			row.Status = DeliveryFailed
			row.Attempt++
			row.LastError = sanitizeDetails(err.Error())
			row.NextRunAt = time.Now().UTC().Add(retryBackoff(row.Attempt))
			_ = s.DB.WithContext(ctx).Save(&row).Error
			continue
		}
		now := time.Now().UTC()
		row.Status = DeliveryDelivered
		row.DeliveredAt = &now
		row.LastError = ""
		_ = s.DB.WithContext(ctx).Save(&row).Error
	}
	return nil
}

func (s *Service) channelByName(name string) Channel {
	for _, ch := range s.Channels {
		if ch != nil && ch.Name() == name {
			return ch
		}
	}
	return nil
}

// StartEvaluatorWorker periodically evaluates rules with an external sample source.
func StartEvaluatorWorker(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger, svc *Service, interval time.Duration, sample func() metrics.Snapshot) {
	if wg == nil || svc == nil || sample == nil {
		return
	}
	if interval <= 0 {
		interval = time.Minute
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if _, err := svc.EvaluateSnapshot(ctx, sample()); err != nil && log != nil {
					log.Warn("alert_evaluator_failed", "error", err)
				}
			}
		}
	}()
}

// StartDeliveryWorker periodically retries failed/pending alert deliveries.
func StartDeliveryWorker(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger, svc *Service, interval time.Duration) {
	if wg == nil || svc == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if err := svc.DeliverPending(ctx, 50); err != nil && log != nil && err != gorm.ErrRecordNotFound {
					log.Warn("alert_delivery_failed", "error", err)
				}
			}
		}
	}()
}

func compare(value float64, condition string, threshold float64) bool {
	switch condition {
	case ">", "gt", "increase", "ratio", "p95", "gauge_max":
		return value > threshold
	case ">=", "gte":
		return value >= threshold
	case "<", "lt":
		return value < threshold
	case "<=", "lte":
		return value <= threshold
	default:
		return value > threshold
	}
}

func metricModule(metric string) string {
	switch {
	case strings.HasPrefix(metric, "http_"):
		return "http"
	case strings.HasPrefix(metric, "provider_"), strings.HasPrefix(metric, "ai_"):
		return "provider"
	case strings.HasPrefix(metric, "task_"), strings.HasPrefix(metric, "tasks_"):
		return "task"
	case strings.HasPrefix(metric, "webhook_"):
		return "webhook"
	case strings.HasPrefix(metric, "auth_"), strings.HasPrefix(metric, "audit_"), strings.HasPrefix(metric, "tenant_"):
		return "security"
	default:
		return "observability"
	}
}

func retryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(attempt*attempt) * time.Second
}
