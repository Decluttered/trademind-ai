package alerting

import (
	"fmt"
	"strings"
	"time"

	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
)

type evaluationKind string

const (
	evaluationCounter   evaluationKind = "counter_increase"
	evaluationRatio     evaluationKind = "counter_ratio"
	evaluationHistogram evaluationKind = "histogram_p95"
	evaluationGauge     evaluationKind = "gauge_max"
	evaluationAge       evaluationKind = "timestamp_age"
)

type metricSelector struct {
	Name   string
	Labels map[string]string
}

type evaluationSpec struct {
	Kind        evaluationKind
	Numerator   metricSelector
	Denominator metricSelector
	Minimum     uint64
}

var defaultEvaluationSpecs = map[string]evaluationSpec{
	"audit_chain_mismatch":        {Kind: evaluationCounter, Numerator: metricSelector{Name: "audit_chain_mismatch_total"}},
	"auth_refresh_reuse":          {Kind: evaluationCounter, Numerator: metricSelector{Name: "auth_refresh_reuse_detected_total"}},
	"http_5xx_elevated":           {Kind: evaluationRatio, Numerator: metricSelector{Name: "http_server_requests_total", Labels: map[string]string{"status_class": "5xx"}}, Denominator: metricSelector{Name: "http_server_requests_total"}, Minimum: 20},
	"provider_timeout_elevated":   {Kind: evaluationRatio, Numerator: metricSelector{Name: "provider_request_timeouts_total"}, Denominator: metricSelector{Name: "provider_requests_total"}, Minimum: 20},
	"ai_image_provider_timeout":   {Kind: evaluationCounter, Numerator: metricSelector{Name: "ai_image_provider_timeouts_total"}},
	"task_dead_letter_spike":      {Kind: evaluationCounter, Numerator: metricSelector{Name: "tasks_dead_letter_total"}},
	"task_queue_backlog":          {Kind: evaluationHistogram, Numerator: metricSelector{Name: "task_queue_age_seconds"}, Minimum: 1},
	"webhook_lag":                 {Kind: evaluationHistogram, Numerator: metricSelector{Name: "webhook_processing_lag_seconds"}, Minimum: 1},
	"order_sync_lag":              {Kind: evaluationHistogram, Numerator: metricSelector{Name: "order_sync_cursor_lag_seconds"}, Minimum: 1},
	"file_scan_backlog":           {Kind: evaluationHistogram, Numerator: metricSelector{Name: "file_scan_queue_age_seconds"}, Minimum: 1},
	"secret_rotation_failed":      {Kind: evaluationCounter, Numerator: metricSelector{Name: "secret_rotation_failures_total"}},
	"tenant_access_denial_spike":  {Kind: evaluationRatio, Numerator: metricSelector{Name: "tenant_access_denied_total"}, Denominator: metricSelector{Name: "http_server_requests_total"}, Minimum: 100},
	"backup_consecutive_failures": {Kind: evaluationCounter, Numerator: metricSelector{Name: "backup_failures_total", Labels: map[string]string{"result": "failure"}}},
	"backup_too_old":              {Kind: evaluationAge, Numerator: metricSelector{Name: "backup_last_success_timestamp"}},
	"backup_verification_failed":  {Kind: evaluationCounter, Numerator: metricSelector{Name: "backup_verification_total", Labels: map[string]string{"result": "failure"}}},
	"restore_validation_failed":   {Kind: evaluationCounter, Numerator: metricSelector{Name: "restore_validation_total", Labels: map[string]string{"result": "failure"}}},
}

func evaluateRule(rule AlertRule, history []metrics.Snapshot, current metrics.Snapshot) (float64, uint64, bool, error) {
	spec, ok := defaultEvaluationSpecs[rule.ID]
	if !ok {
		return 0, 0, false, fmt.Errorf("alert rule %s has no supported evaluation spec", rule.ID)
	}
	if spec.Kind == evaluationGauge {
		value, ready := metrics.GaugeMax(current, spec.Numerator.Name, spec.Numerator.Labels)
		return value, 1, ready, nil
	}
	if spec.Kind == evaluationAge {
		lastSuccess, ready := metrics.GaugeMax(current, spec.Numerator.Name, spec.Numerator.Labels)
		if !ready || lastSuccess <= 0 {
			return 0, 0, false, nil
		}
		age := current.TakenAt.Sub(time.Unix(int64(lastSuccess), 0))
		if age < 0 {
			return 0, 0, false, nil
		}
		return age.Seconds(), 1, true, nil
	}
	window, err := time.ParseDuration(strings.TrimSpace(rule.Window))
	if err != nil || window <= 0 {
		return 0, 0, false, fmt.Errorf("alert rule %s has invalid window %q", rule.ID, rule.Window)
	}
	baseline, ready := windowBaseline(history, current.TakenAt.Add(-window))
	if !ready {
		return 0, 0, false, nil
	}
	switch spec.Kind {
	case evaluationCounter:
		value, ok := metrics.CounterDelta(baseline, current, spec.Numerator.Name, spec.Numerator.Labels)
		return value, uint64(value), ok, nil
	case evaluationRatio:
		numerator, numeratorOK := metrics.CounterDelta(baseline, current, spec.Numerator.Name, spec.Numerator.Labels)
		denominator, denominatorOK := metrics.CounterDelta(baseline, current, spec.Denominator.Name, spec.Denominator.Labels)
		if !numeratorOK || !denominatorOK || denominator < float64(spec.Minimum) {
			return 0, uint64(denominator), false, nil
		}
		return numerator / denominator, uint64(denominator), true, nil
	case evaluationHistogram:
		value, count, ok := metrics.HistogramQuantileDelta(baseline, current, spec.Numerator.Name, spec.Numerator.Labels, 0.95)
		if !ok || count < spec.Minimum {
			return 0, count, false, nil
		}
		return value, count, true, nil
	default:
		return 0, 0, false, fmt.Errorf("alert rule %s uses unsupported evaluation kind %q", rule.ID, spec.Kind)
	}
}

func windowBaseline(history []metrics.Snapshot, target time.Time) (metrics.Snapshot, bool) {
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
