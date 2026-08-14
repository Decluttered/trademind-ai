package observabilitymod

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/trademind-ai/trademind/backend/internal/database"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"gorm.io/gorm"
)

const (
	SLOStatusAchieved         = "achieved"
	SLOStatusViolated         = "violated"
	SLOStatusInsufficientData = "insufficient_data"
)

// EnsureDefaultSLOs seeds code-level SLO definitions idempotently.
func EnsureDefaultSLOs(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	defs := []database.SLODefinition{
		{ID: "api_availability", Name: "API availability", TargetRatio: 0.995, Window: "1h", Enabled: true},
		{ID: "api_latency", Name: "API latency", TargetRatio: 0.99, Window: "1h", Enabled: true},
		{ID: "worker_success", Name: "Worker success", TargetRatio: 0.99, Window: "1h", Enabled: true},
		{ID: "webhook_processing", Name: "Webhook processing", TargetRatio: 0.99, Window: "1h", Enabled: true},
		{ID: "provider_success", Name: "Provider success", TargetRatio: 0.98, Window: "1h", Enabled: true},
		{ID: "order_sync_freshness", Name: "Order sync freshness", TargetRatio: 0.98, Window: "1h", Enabled: true},
		{ID: "file_scan_completion", Name: "File scan completion", TargetRatio: 0.99, Window: "1h", Enabled: true},
		{ID: "audit_write_success", Name: "Audit write success", TargetRatio: 0.999, Window: "1h", Enabled: true},
	}
	for _, def := range defs {
		var count int64
		if err := db.WithContext(ctx).Model(&database.SLODefinition{}).Where("id = ?", def.ID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.WithContext(ctx).Create(&def).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// EvaluateSLOs writes snapshots for enabled SLOs from aggregate samples.
func EvaluateSLOs(ctx context.Context, db *gorm.DB, cat *metrics.Catalog, samples map[string]float64) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("slo evaluator unavailable")
	}
	var defs []database.SLODefinition
	if err := db.WithContext(ctx).Where("enabled = ?", true).Find(&defs).Error; err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	written := 0
	for _, def := range defs {
		total, errors := sloInputs(def.ID, samples)
		compliance, remaining, burn, status := calculateSLO(total, errors, def.TargetRatio)
		snap := database.SLOSnapshot{
			SLOID:       def.ID,
			Compliance:  compliance,
			ErrorBudget: remaining,
			BurnRate:    burn,
			Window:      normalizeWindow(def.Window),
			Status:      status,
			RecordedAt:  now,
		}
		if err := db.WithContext(ctx).Create(&snap).Error; err != nil {
			return written, err
		}
		if cat != nil {
			cat.ObserveSLO(def.ID, snap.Window, compliance, remaining, burn)
		}
		written++
	}
	return written, nil
}

func StartSLOEvaluatorWorker(ctx context.Context, wg *sync.WaitGroup, log *slog.Logger, db *gorm.DB, cat *metrics.Catalog, interval time.Duration, sample func() metrics.Snapshot) {
	if wg == nil || db == nil || sample == nil {
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
		history := make([]metrics.Snapshot, 0, 64)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				current := sample()
				history = appendSLOSnapshot(history, current)
				if n, err := EvaluateSLOWindow(ctx, db, cat, history, current); err != nil && log != nil {
					log.Warn("slo_evaluator_failed", "error", err)
				} else if log != nil {
					log.Debug("slo_evaluator_completed", "snapshots", n)
				}
			}
		}
	}()
}

// EvaluateSLOWindow evaluates configured SLOs from counter and histogram
// increases within their declared windows.
func EvaluateSLOWindow(ctx context.Context, db *gorm.DB, cat *metrics.Catalog, history []metrics.Snapshot, current metrics.Snapshot) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("slo evaluator unavailable")
	}
	var defs []database.SLODefinition
	if err := db.WithContext(ctx).Where("enabled = ?", true).Find(&defs).Error; err != nil {
		return 0, err
	}
	written := 0
	for _, def := range defs {
		window := sloWindowDuration(def.Window)
		baseline, ready := sloBaseline(history, current.TakenAt.Add(-window))
		if !ready {
			continue
		}
		total, errors, dataReady := sloWindowInputs(def.ID, baseline, current)
		if !dataReady {
			continue
		}
		compliance, remaining, burn, status := calculateSLO(total, errors, def.TargetRatio)
		snapshot := database.SLOSnapshot{
			SLOID: def.ID, Compliance: compliance, ErrorBudget: remaining, BurnRate: burn,
			Window: normalizeWindow(def.Window), Status: status, RecordedAt: current.TakenAt.UTC().Unix(),
		}
		if err := db.WithContext(ctx).Create(&snapshot).Error; err != nil {
			return written, err
		}
		if cat != nil {
			cat.ObserveSLO(def.ID, snapshot.Window, compliance, remaining, burn)
		}
		written++
	}
	return written, nil
}

func appendSLOSnapshot(history []metrics.Snapshot, current metrics.Snapshot) []metrics.Snapshot {
	cutoff := current.TakenAt.Add(-30 * 24 * time.Hour)
	recentCutoff := current.TakenAt.Add(-2 * time.Hour)
	next := make([]metrics.Snapshot, 0, len(history)+1)
	olderHours := make(map[int64]int)
	for _, snapshot := range history {
		if snapshot.TakenAt.Before(cutoff) || !snapshot.TakenAt.Before(current.TakenAt) {
			continue
		}
		if snapshot.TakenAt.Before(recentCutoff) {
			hour := snapshot.TakenAt.UTC().Truncate(time.Hour).Unix()
			if index, exists := olderHours[hour]; exists {
				next[index] = snapshot
				continue
			}
			olderHours[hour] = len(next)
		}
		next = append(next, snapshot)
	}
	next = append(next, current)
	if len(next) > 900 {
		next = next[len(next)-900:]
	}
	return next
}

func sloBaseline(history []metrics.Snapshot, target time.Time) (metrics.Snapshot, bool) {
	var baseline metrics.Snapshot
	found := false
	for _, snapshot := range history {
		if snapshot.TakenAt.After(target) {
			continue
		}
		if !found || snapshot.TakenAt.After(baseline.TakenAt) {
			baseline = snapshot
			found = true
		}
	}
	return baseline, found
}

func sloWindowInputs(id string, previous, current metrics.Snapshot) (float64, float64, bool) {
	delta := func(name string, labels map[string]string) (float64, bool) {
		return metrics.CounterDelta(previous, current, name, labels)
	}
	sum := func(selectors ...struct {
		name   string
		labels map[string]string
	}) (float64, bool) {
		total := 0.0
		for _, selector := range selectors {
			value, ok := delta(selector.name, selector.labels)
			if !ok {
				return 0, false
			}
			total += value
		}
		return total, true
	}
	switch id {
	case "api_availability":
		total, totalOK := delta("http_server_requests_total", nil)
		errors, errorsOK := delta("http_server_requests_total", map[string]string{"status_class": "5xx"})
		return total, errors, totalOK && errorsOK
	case "api_latency":
		within, total, ok := metrics.HistogramThresholdDelta(previous, current, "http_server_request_duration_seconds", nil, 1)
		return float64(total), float64(total - within), ok
	case "worker_success":
		completed, completedOK := delta("tasks_completed_total", nil)
		failed, failedOK := delta("tasks_failed_total", nil)
		dead, deadOK := delta("tasks_dead_letter_total", nil)
		return completed + failed + dead, failed + dead, completedOK && failedOK && deadOK
	case "webhook_processing":
		processed, processedOK := delta("webhook_events_processed_total", nil)
		rejected, rejectedOK := delta("webhook_payload_rejected_total", nil)
		resolution, resolutionOK := delta("webhook_shop_resolution_failures_total", nil)
		return processed + rejected, rejected + resolution, processedOK && rejectedOK && resolutionOK
	case "provider_success":
		total, totalOK := delta("provider_requests_total", nil)
		timeouts, timeoutsOK := delta("provider_request_timeouts_total", nil)
		contract, contractOK := delta("provider_contract_mismatches_total", nil)
		return total, timeouts + contract, totalOK && timeoutsOK && contractOK
	case "order_sync_freshness":
		total, totalOK := delta("order_sync_runs_total", nil)
		errors, errorsOK := delta("order_sync_failures_total", nil)
		return total, errors, totalOK && errorsOK
	case "file_scan_completion":
		total, totalOK := delta("file_scan_tasks_total", nil)
		errors, errorsOK := sum(
			struct {
				name   string
				labels map[string]string
			}{"file_scan_failures_total", nil},
			struct {
				name   string
				labels map[string]string
			}{"file_scan_stuck_total", nil},
		)
		return total, errors, totalOK && errorsOK
	case "audit_write_success":
		total, totalOK := delta("security_events_total", nil)
		errors, errorsOK := delta("audit_chain_mismatch_total", nil)
		return total + errors, errors, totalOK && errorsOK
	default:
		return 0, 0, false
	}
}

func sloWindowDuration(raw string) time.Duration {
	switch normalizeWindow(raw) {
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func sloInputs(id string, samples map[string]float64) (float64, float64) {
	if samples == nil {
		return 0, 0
	}
	if total := samples["slo:"+id+":total"]; total > 0 {
		return total, samples["slo:"+id+":errors"]
	}
	switch id {
	case "api_availability", "api_latency":
		return samples["http_server_requests_total"], samples["http_server_panics_total"]
	case "worker_success":
		total := samples["tasks_completed_total"] + samples["tasks_failed_total"] + samples["tasks_dead_letter_total"]
		return total, samples["tasks_failed_total"] + samples["tasks_dead_letter_total"]
	case "webhook_processing":
		total := samples["webhook_events_processed_total"] + samples["webhook_payload_rejected_total"]
		return total, samples["webhook_payload_rejected_total"] + samples["webhook_shop_resolution_failures_total"]
	case "provider_success":
		return samples["provider_requests_total"], samples["provider_request_timeouts_total"] + samples["provider_contract_mismatches_total"]
	case "order_sync_freshness":
		return samples["order_sync_runs_total"], samples["order_sync_failures_total"]
	case "file_scan_completion":
		return samples["file_scan_tasks_total"], samples["file_scan_failures_total"] + samples["file_scan_stuck_total"]
	case "audit_write_success":
		total := samples["security_events_total"] + samples["audit_chain_mismatch_total"]
		return total, samples["audit_chain_mismatch_total"]
	default:
		return 0, 0
	}
}

func calculateSLO(total, errors, target float64) (float64, float64, float64, string) {
	if total <= 0 || target <= 0 || target >= 1 {
		return 0, 0, 0, SLOStatusInsufficientData
	}
	if errors < 0 {
		errors = 0
	}
	if errors > total {
		errors = total
	}
	errorRatio := errors / total
	compliance := 1 - errorRatio
	allowed := 1 - target
	remaining := (allowed - errorRatio) / allowed
	remaining = math.Max(0, math.Min(1, remaining))
	burn := errorRatio / allowed
	status := SLOStatusAchieved
	if compliance < target {
		status = SLOStatusViolated
	}
	return compliance, remaining, burn, status
}

func normalizeWindow(raw string) string {
	switch raw {
	case "1h", "6h", "24h", "7d", "30d":
		return raw
	default:
		return "1h"
	}
}
