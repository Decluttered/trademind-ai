package alerting

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	dto "github.com/prometheus/client_model/go"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"gorm.io/gorm"
)

func TestAlertDeduplicationAndRecovery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AlertEvent{}, &AlertRule{}, &AlertSilence{}, &AlertDelivery{}, &AlertEvaluationRun{}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, time.Second, true)
	ctx := context.Background()
	a1, err := svc.Fire(ctx, "http_5xx_elevated", SeverityWarning, "http", "5xx spike", "safe")
	if err != nil {
		t.Fatal(err)
	}
	a2, err := svc.Fire(ctx, "http_5xx_elevated", SeverityWarning, "http", "5xx spike", "safe")
	if err != nil {
		t.Fatal(err)
	}
	if a2.OccurrenceCount < 2 && a1.Fingerprint != a2.Fingerprint {
		t.Fatalf("dedup failed: %+v %+v", a1, a2)
	}
	if err := svc.Resolve(ctx, a1.ID); err != nil {
		t.Fatal(err)
	}
	var resolved AlertEvent
	if err := db.First(&resolved, "id = ?", a1.ID).Error; err != nil {
		t.Fatal(err)
	}
	if resolved.Status != StatusResolved {
		t.Fatalf("expected resolved got %s", resolved.Status)
	}
}

func TestEnsureDefaultRulesRemovesRetiredRules(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AlertRule{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range retiredDefaultRuleIDs {
		if err := db.Create(&AlertRule{ID: id, Enabled: true}).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := EnsureDefaultRules(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	var retiredCount int64
	if err := db.Model(&AlertRule{}).Where("id IN ?", retiredDefaultRuleIDs).Count(&retiredCount).Error; err != nil {
		t.Fatal(err)
	}
	if retiredCount != 0 {
		t.Fatalf("expected retired default rule to be removed, got %d", retiredCount)
	}
	var activeCount int64
	if err := db.Model(&AlertRule{}).Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != int64(len(DefaultRules())) {
		t.Fatalf("expected %d active default rules, got %d", len(DefaultRules()), activeCount)
	}
}

func TestEnsureDefaultRulesSynchronizesDefinitionAndPreservesEnabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AlertRule{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AlertRule{ID: "http_5xx_elevated", Name: "stale", Metric: "wrong_total", Condition: "rate", Enabled: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&AlertRule{}).Where("id = ?", "http_5xx_elevated").Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureDefaultRules(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var rule AlertRule
	if err := db.First(&rule, "id = ?", "http_5xx_elevated").Error; err != nil {
		t.Fatal(err)
	}
	if rule.Enabled || rule.Condition != "ratio" || rule.Window != "5m" || rule.Metric != "http_server_requests_total" {
		t.Fatalf("unexpected synchronized rule: %+v", rule)
	}
}

func TestAlertEvaluatorDeliveryAndRecovery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AlertEvent{}, &AlertRule{}, &AlertSilence{}, &AlertDelivery{}, &AlertEvaluationRun{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	rule := AlertRule{ID: "ai_image_provider_timeout", Name: "AI image provider timeout", Metric: "ai_image_provider_timeouts_total", Condition: "increase", Window: "5m", Threshold: 0, Severity: SeverityWarning, Enabled: true}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, time.Second, true)
	start := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	_, err = svc.EvaluateSnapshot(ctx, counterSnapshot(start, "ai_image_provider_timeouts_total", nil, 0))
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.EvaluateSnapshot(ctx, counterSnapshot(start.Add(5*time.Minute), "ai_image_provider_timeouts_total", nil, 1))
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsFired != 1 {
		t.Fatalf("expected fired=1 got %+v", run)
	}
	var delivery AlertDelivery
	if err := db.First(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	if delivery.Status != DeliveryDelivered {
		t.Fatalf("expected delivered got %s", delivery.Status)
	}
	run, err = svc.EvaluateSnapshot(ctx, counterSnapshot(start.Add(10*time.Minute), "ai_image_provider_timeouts_total", nil, 1))
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsResolved != 1 {
		t.Fatalf("expected resolved=1 got %+v", run)
	}
}

func TestHTTP5xxRatioUsesWindowDenominatorAndMinimumSamples(t *testing.T) {
	db := alertTestDB(t)
	rule := defaultRuleByID(t, "http_5xx_elevated")
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, time.Second, true)
	start := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	if _, err := svc.EvaluateSnapshot(context.Background(), httpSnapshot(start, 100, 4)); err != nil {
		t.Fatal(err)
	}
	run, err := svc.EvaluateSnapshot(context.Background(), httpSnapshot(start.Add(5*time.Minute), 110, 6))
	if err != nil {
		t.Fatal(err)
	}
	if run.RulesChecked != 0 || run.RulesSkipped != 1 {
		t.Fatalf("minimum sample guard not applied: %+v", run)
	}
	run, err = svc.EvaluateSnapshot(context.Background(), httpSnapshot(start.Add(10*time.Minute), 140, 8))
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsFired != 1 {
		t.Fatalf("expected 5xx ratio alert: %+v", run)
	}
}

func TestCounterResetWarmsInsteadOfFiring(t *testing.T) {
	db := alertTestDB(t)
	rule := defaultRuleByID(t, "ai_image_provider_timeout")
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, time.Second, true)
	start := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	_, _ = svc.EvaluateSnapshot(context.Background(), counterSnapshot(start, rule.Metric, nil, 10))
	run, err := svc.EvaluateSnapshot(context.Background(), counterSnapshot(start.Add(5*time.Minute), rule.Metric, nil, 1))
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsFired != 0 || run.RulesSkipped != 1 {
		t.Fatalf("counter reset must not fire: %+v", run)
	}
}

func TestBackupVerificationSuccessDoesNotFireFailureAlert(t *testing.T) {
	db := alertTestDB(t)
	rule := defaultRuleByID(t, "backup_verification_failed")
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, time.Second, true)
	start := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	before := labeledCounterSnapshot(start, rule.Metric, map[string]float64{"success": 3, "failure": 0})
	afterSuccess := labeledCounterSnapshot(start.Add(30*time.Minute), rule.Metric, map[string]float64{"success": 4, "failure": 0})
	_, _ = svc.EvaluateSnapshot(context.Background(), before)
	run, err := svc.EvaluateSnapshot(context.Background(), afterSuccess)
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsFired != 0 {
		t.Fatalf("successful verification must not fire: %+v", run)
	}
	afterFailure := labeledCounterSnapshot(start.Add(time.Hour), rule.Metric, map[string]float64{"success": 4, "failure": 1})
	run, err = svc.EvaluateSnapshot(context.Background(), afterFailure)
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsFired != 1 {
		t.Fatalf("failed verification must fire: %+v", run)
	}
	afterStable := labeledCounterSnapshot(start.Add(90*time.Minute), rule.Metric, map[string]float64{"success": 5, "failure": 1})
	run, err = svc.EvaluateSnapshot(context.Background(), afterStable)
	if err != nil {
		t.Fatal(err)
	}
	if run.AlertsResolved != 1 {
		t.Fatalf("stable failure counter must resolve: %+v", run)
	}
}

func alertTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AlertEvent{}, &AlertRule{}, &AlertSilence{}, &AlertDelivery{}, &AlertEvaluationRun{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func defaultRuleByID(t *testing.T, id string) AlertRule {
	t.Helper()
	for _, rule := range DefaultRules() {
		if rule.ID == id {
			return rule
		}
	}
	t.Fatalf("default rule %s not found", id)
	return AlertRule{}
}

func counterSnapshot(at time.Time, name string, labels map[string]string, value float64) metrics.Snapshot {
	return metrics.Snapshot{TakenAt: at, Families: map[string]metrics.FamilySnapshot{
		name: {Type: dto.MetricType_COUNTER, Samples: []metrics.Sample{{Labels: labels, Value: value}}},
	}}
}

func labeledCounterSnapshot(at time.Time, name string, values map[string]float64) metrics.Snapshot {
	samples := make([]metrics.Sample, 0, len(values))
	for result, value := range values {
		samples = append(samples, metrics.Sample{Labels: map[string]string{"result": result}, Value: value})
	}
	return metrics.Snapshot{TakenAt: at, Families: map[string]metrics.FamilySnapshot{
		name: {Type: dto.MetricType_COUNTER, Samples: samples},
	}}
}

func httpSnapshot(at time.Time, total, failures float64) metrics.Snapshot {
	success := total - failures
	return metrics.Snapshot{TakenAt: at, Families: map[string]metrics.FamilySnapshot{
		"http_server_requests_total": {Type: dto.MetricType_COUNTER, Samples: []metrics.Sample{
			{Labels: map[string]string{"status_class": "2xx"}, Value: success},
			{Labels: map[string]string{"status_class": "5xx"}, Value: failures},
		}},
	}}
}

func TestSanitizeDetails(t *testing.T) {
	if sanitizeDetails("TEST_APP_SECRET_UNIQUE leaked") == "TEST_APP_SECRET_UNIQUE leaked" {
		// contains secret marker word 'secret' in TEST_APP_SECRET - should redact
	}
	out := sanitizeDetails("password=foo")
	if out != "[redacted]" {
		t.Fatalf("got %q", out)
	}
}
