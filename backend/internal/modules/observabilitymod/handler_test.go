package observabilitymod

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/database"
	"github.com/trademind-ai/trademind/backend/internal/modules/alerting"
	"github.com/trademind-ai/trademind/backend/internal/pkg/metrics"
	"github.com/trademind-ai/trademind/backend/internal/pkg/observability"
	"gorm.io/gorm"
)

func TestOverviewReturnsOperationalSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&alerting.AlertEvent{}, &alerting.AlertEvaluationRun{}, &database.SLODefinition{}, &database.SLOSnapshot{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	if err := db.Create(&alerting.AlertEvent{ID: "critical-1", Status: alerting.StatusFiring, Severity: alerting.SeverityCritical}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&alerting.AlertEvaluationRun{ID: "run-1", StartedAt: now, Status: alerting.EvaluationSucceeded, RulesChecked: 12, RulesSkipped: 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.SLODefinition{ID: "api_availability", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.SLOSnapshot{SLOID: "api_availability", Status: SLOStatusAchieved, RecordedAt: now.Unix()}).Error; err != nil {
		t.Fatal(err)
	}
	registry := metrics.NewRegistry("test")
	catalog, err := metrics.RegisterCatalog(registry)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		DB: db,
		Cfg: &config.Config{Observability: config.ObservabilityConfig{
			Enabled: true, Mode: config.ObsModeHybrid, Environment: "production",
			MetricsEnabled: true, MetricsPath: "/internal/metrics", MetricsInternalOnly: true,
			MetricsAllowlistCIDRs: []string{"127.0.0.1/32"}, AlertingEnabled: true,
		}},
		Obs: &observability.Observability{Metrics: registry, Catalog: catalog},
	}
	r := gin.New()
	r.GET("/overview", func(c *gin.Context) {
		// Permission middleware is tested separately; inject the endpoint body here.
		obs := h.obsConfig()
		metricsSummary := h.metricsSummary(obs)
		response := gin.H{
			"overallStatus": overallStatus(obs, h.alertSummary(), h.alertEvaluationSummary(), h.sloSummary(), metricsSummary, h.telemetryStatus()),
			"alerts":        h.alertSummary(), "evaluation": h.alertEvaluationSummary(), "slo": h.sloSummary(),
			"metrics": metricsSummary,
		}
		c.JSON(http.StatusOK, response)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/overview", nil)
	r.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["overallStatus"] != "needs_attention" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestRegisterExposesOnlyLiveObservabilitySummaryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r.Group("/api/v1"), &Handler{})

	routes := make(map[string]bool)
	for _, route := range r.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	if !routes[http.MethodGet+" /api/v1/observability/overview"] {
		t.Fatal("observability overview route is not registered")
	}
	for _, path := range []string{"/http", "/tasks", "/providers", "/security"} {
		if routes[http.MethodGet+" /api/v1/observability"+path] {
			t.Fatalf("placeholder route %s is still registered", path)
		}
	}
}

func TestOverallStatusDoesNotReportUnknownOrIncompleteTelemetryAsHealthy(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	healthyAlerts := gin.H{"status": "active", "active": int64(0)}
	succeededEvaluation := gin.H{"status": alerting.EvaluationSucceeded}
	achievedSLO := gin.H{"status": SLOStatusAchieved}
	activeMetrics := gin.H{"status": "active"}
	disabledTelemetry := gin.H{"status": "disabled"}

	tests := []struct {
		name       string
		evaluation gin.H
		slo        gin.H
		metrics    gin.H
		telemetry  gin.H
		want       string
	}{
		{name: "healthy", evaluation: succeededEvaluation, slo: achievedSLO, metrics: activeMetrics, telemetry: disabledTelemetry, want: "healthy"},
		{name: "metrics unavailable", evaluation: succeededEvaluation, slo: achievedSLO, metrics: gin.H{"status": "unavailable"}, telemetry: disabledTelemetry, want: "needs_attention"},
		{name: "metrics unprotected", evaluation: succeededEvaluation, slo: achievedSLO, metrics: gin.H{"status": "unprotected"}, telemetry: disabledTelemetry, want: "needs_attention"},
		{name: "alerts unavailable", evaluation: succeededEvaluation, slo: achievedSLO, metrics: activeMetrics, telemetry: disabledTelemetry, want: "needs_attention"},
		{name: "telemetry incomplete", evaluation: succeededEvaluation, slo: achievedSLO, metrics: activeMetrics, telemetry: gin.H{"status": "incomplete"}, want: "needs_attention"},
		{name: "telemetry pending", evaluation: succeededEvaluation, slo: achievedSLO, metrics: activeMetrics, telemetry: gin.H{"status": "export_pending"}, want: "waiting"},
		{name: "telemetry deferred", evaluation: succeededEvaluation, slo: achievedSLO, metrics: activeMetrics, telemetry: gin.H{"status": "real_backend_deferred"}, want: "waiting"},
		{name: "slo insufficient", evaluation: succeededEvaluation, slo: gin.H{"status": SLOStatusInsufficientData}, metrics: activeMetrics, telemetry: disabledTelemetry, want: "waiting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := healthyAlerts
			if tt.name == "alerts unavailable" {
				alerts = gin.H{"status": "unavailable", "active": int64(0)}
			}
			if got := overallStatus(obs, alerts, tt.evaluation, tt.slo, tt.metrics, tt.telemetry); got != tt.want {
				t.Fatalf("overallStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOperationalSummariesFailClosedWhenDatabaseIsUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		DB: db,
		Cfg: &config.Config{Observability: config.ObservabilityConfig{
			Enabled: true, MetricsEnabled: true, AlertingEnabled: true,
		}},
	}
	if got := h.alertSummary()["status"]; got != "unavailable" {
		t.Fatalf("alert summary status = %v", got)
	}
	if got := h.alertEvaluationSummary()["status"]; got != "unavailable" {
		t.Fatalf("evaluation summary status = %v", got)
	}
	if got := h.sloSummary()["status"]; got != "unavailable" {
		t.Fatalf("SLO summary status = %v", got)
	}
}

func TestSLOSummaryFailsClosedWithoutDatabase(t *testing.T) {
	h := &Handler{Cfg: &config.Config{Observability: config.ObservabilityConfig{
		Enabled: true, MetricsEnabled: true,
	}}}
	if got := h.sloSummary()["status"]; got != "unavailable" {
		t.Fatalf("SLO summary status = %v", got)
	}
}

func TestOTLPExporterStatusPrefersConfigurationErrorsOverPendingState(t *testing.T) {
	h := &Handler{Cfg: &config.Config{Observability: config.ObservabilityConfig{
		TracingEnabled: true, OTELExporterOTLPEndpoint: "http://collector:4318", OTELExporterOTLPProtocol: "grpc",
	}}}
	if got := h.otlpExporterStatus(); got != "incomplete" {
		t.Fatalf("otlpExporterStatus() = %q, want incomplete", got)
	}
}

func TestSLOSummaryUsesWorstLatestStatusAcrossEnabledDefinitions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&database.SLODefinition{}, &database.SLOSnapshot{}); err != nil {
		t.Fatal(err)
	}
	definitions := []database.SLODefinition{
		{ID: "api_availability", Enabled: true},
		{ID: "worker_success", Enabled: true},
		{ID: "disabled_history", Enabled: false},
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&database.SLODefinition{}).Where("id = ?", "disabled_history").Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	snapshots := []database.SLOSnapshot{
		{SLOID: "api_availability", Status: SLOStatusAchieved, RecordedAt: 100},
		{SLOID: "worker_success", Status: SLOStatusAchieved, RecordedAt: 90},
		{SLOID: "worker_success", Status: SLOStatusViolated, RecordedAt: 110},
		{SLOID: "disabled_history", Status: SLOStatusViolated, RecordedAt: 120},
	}
	if err := db.Create(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		DB: db,
		Cfg: &config.Config{Observability: config.ObservabilityConfig{
			Enabled: true, MetricsEnabled: true,
		}},
	}
	summary := h.sloSummary()
	if summary["status"] != SLOStatusViolated {
		t.Fatalf("unexpected SLO summary: %+v", summary)
	}
	if summary["lastEvaluatedAt"] != time.Unix(110, 0).UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected latest evaluation: %+v", summary)
	}
}
