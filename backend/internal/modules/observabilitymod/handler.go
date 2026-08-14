package observabilitymod

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/database"
	"github.com/trademind-ai/trademind/backend/internal/modules/alerting"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"github.com/trademind-ai/trademind/backend/internal/pkg/adminperm"
	"github.com/trademind-ai/trademind/backend/internal/pkg/observability"
	"github.com/trademind-ai/trademind/backend/internal/pkg/response"
	"gorm.io/gorm"
)

// Handler serves observability aggregation APIs.
type Handler struct {
	DB    *gorm.DB
	Cfg   *config.Config
	Obs   *observability.Observability
	Alert *alerting.Service
	OpLog *operationlog.Service
}

// Register mounts observability routes.
func Register(r gin.IRouter, h *Handler) {
	if r == nil || h == nil {
		return
	}
	g := r.Group("/observability")
	g.GET("/overview", h.Overview)
	alerting.Register(r, &alerting.Handler{Svc: h.Alert, OpLog: h.OpLog})
}

func (h *Handler) requireRead(c *gin.Context) bool {
	return adminperm.RequirePermission(c, h.DB, adminperm.PermObservabilityRead)
}

// Overview returns system observability summary.
func (h *Handler) Overview(c *gin.Context) {
	if !h.requireRead(c) {
		return
	}
	obs := h.obsConfig()
	alerts := h.alertSummary()
	evaluation := h.alertEvaluationSummary()
	slo := h.sloSummary()
	metrics := h.metricsSummary(obs)
	telemetry := h.telemetryStatus()
	response.OK(c, gin.H{
		"overallStatus":     overallStatus(obs, alerts, evaluation, slo, metrics, telemetry),
		"enabled":           obs.Enabled,
		"mode":              obs.Mode,
		"metricsEnabled":    obs.MetricsEnabled,
		"tracingEnabled":    obs.TracingEnabled,
		"alertingEnabled":   obs.AlertingEnabled,
		"metricsPath":       obs.MetricsPath,
		"metricsInternal":   obs.MetricsInternalOnly,
		"otelExportBlocked": h.exportBlocked(),
		"runtimeStatus":     h.runtimeStatus(),
		"metrics":           metrics,
		"alerts":            alerts,
		"evaluation":        evaluation,
		"slo":               slo,
		"telemetry":         telemetry,
		"environment":       obs.Environment,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) obsConfig() config.ObservabilityConfig {
	if h != nil && h.Cfg != nil {
		return h.Cfg.Observability
	}
	return config.ObservabilityConfig{}
}

func (h *Handler) exportBlocked() bool {
	obs := h.obsConfig()
	if !obs.TracingEnabled || strings.TrimSpace(obs.OTELExporterOTLPEndpoint) == "" {
		return false
	}
	if h != nil && h.Obs != nil && h.Obs.Tracer != nil {
		return h.Obs.Tracer.ExportBlocked()
	}
	return true
}

func (h *Handler) runtimeStatus() gin.H {
	status := gin.H{
		"metricsRegistry":         "deferred",
		"businessInstrumentation": "deferred",
		"otlpExporter":            "deferred",
		"alertEvaluator":          "deferred",
		"alertDelivery":           "deferred",
		"sloEvaluator":            "deferred",
	}
	if h == nil {
		return status
	}
	if h.Obs != nil && h.Obs.Metrics != nil {
		status["metricsRegistry"] = "active"
	}
	status["otlpExporter"] = h.otlpExporterStatus()
	status["otlpProtocol"] = h.obsConfig().OTELExporterOTLPProtocol
	if h.DB != nil {
		var eval alerting.AlertEvaluationRun
		if err := h.DB.Order("started_at DESC").First(&eval).Error; err == nil {
			status["alertEvaluator"] = "active"
			status["lastAlertEvaluationAt"] = eval.StartedAt.UTC().Format(time.RFC3339)
			status["lastAlertEvaluationStatus"] = eval.Status
		}
		var delivery alerting.AlertDelivery
		if err := h.DB.Order("created_at DESC").First(&delivery).Error; err == nil {
			status["alertDelivery"] = "active"
			status["lastAlertDeliveryAt"] = delivery.CreatedAt.UTC().Format(time.RFC3339)
			status["lastAlertDeliveryStatus"] = delivery.Status
		}
		var snap database.SLOSnapshot
		if err := h.DB.Order("recorded_at DESC").First(&snap).Error; err == nil {
			status["sloEvaluator"] = "active"
			status["lastSLOSnapshotAt"] = time.Unix(snap.RecordedAt, 0).UTC().Format(time.RFC3339)
			status["lastSLOStatus"] = snap.Status
		}
	}
	return status
}

func (h *Handler) telemetryStatus() gin.H {
	out := gin.H{
		"status": h.otlpExporterStatus(), "protocol": h.obsConfig().OTELExporterOTLPProtocol,
		"dropped": 0, "exportFailures": 0, "exportSuccess": 0,
	}
	if h == nil || h.Obs == nil || h.Obs.Metrics == nil {
		return out
	}
	values := h.Obs.Metrics.SnapshotValues()
	out["dropped"] = values["telemetry_dropped_items_total"]
	out["exportFailures"] = values["telemetry_export_failures_total"]
	out["exportSuccess"] = values["telemetry_export_success_total"]
	return out
}

func (h *Handler) metricsStatus() string {
	obs := h.obsConfig()
	if !obs.Enabled || !obs.MetricsEnabled {
		return "disabled"
	}
	if !obs.MetricsInternalOnly || len(obs.MetricsAllowlistCIDRs) == 0 {
		return "unprotected"
	}
	if h != nil && h.Obs != nil && h.Obs.Metrics != nil {
		return "active"
	}
	return "unavailable"
}

func (h *Handler) metricsSummary(obs config.ObservabilityConfig) gin.H {
	return gin.H{
		"status":              h.metricsStatus(),
		"path":                obs.MetricsPath,
		"internalOnly":        obs.MetricsInternalOnly,
		"allowlistConfigured": len(obs.MetricsAllowlistCIDRs) > 0,
	}
}

func (h *Handler) alertSummary() gin.H {
	out := gin.H{"status": "active", "active": int64(0), "critical": int64(0), "warning": int64(0)}
	if !h.obsConfig().AlertingEnabled {
		out["status"] = "disabled"
		return out
	}
	if h == nil || h.DB == nil {
		out["status"] = "unavailable"
		return out
	}
	type severityCount struct {
		Severity string
		Count    int64
	}
	var rows []severityCount
	if err := h.DB.Model(&alerting.AlertEvent{}).
		Select("severity, count(*) AS count").
		Where("status IN ?", []string{alerting.StatusFiring, alerting.StatusAcknowledged}).
		Group("severity").Scan(&rows).Error; err != nil {
		out["status"] = "unavailable"
		return out
	}
	active := int64(0)
	for _, row := range rows {
		active += row.Count
		switch row.Severity {
		case alerting.SeverityCritical:
			out["critical"] = row.Count
		case alerting.SeverityWarning:
			out["warning"] = row.Count
		}
	}
	out["active"] = active
	return out
}

func (h *Handler) alertEvaluationSummary() gin.H {
	out := gin.H{"status": "waiting", "rulesChecked": 0, "rulesSkipped": 0, "alertsFired": 0, "alertsResolved": 0}
	if !h.obsConfig().AlertingEnabled {
		out["status"] = "disabled"
		return out
	}
	if h == nil || h.DB == nil {
		out["status"] = "unavailable"
		return out
	}
	var run alerting.AlertEvaluationRun
	if err := h.DB.Order("started_at DESC").First(&run).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			out["status"] = "unavailable"
		}
		return out
	}
	out["status"] = run.Status
	out["lastEvaluatedAt"] = run.StartedAt.UTC().Format(time.RFC3339)
	out["rulesChecked"] = run.RulesChecked
	out["rulesSkipped"] = run.RulesSkipped
	out["alertsFired"] = run.AlertsFired
	out["alertsResolved"] = run.AlertsResolved
	return out
}

func (h *Handler) sloSummary() gin.H {
	out := gin.H{"status": "waiting"}
	if h == nil || !h.obsConfig().Enabled || !h.obsConfig().MetricsEnabled {
		out["status"] = "disabled"
		return out
	}
	if h.DB == nil {
		out["status"] = "unavailable"
		return out
	}
	var definitions []database.SLODefinition
	if err := h.DB.Where("enabled = ?", true).Find(&definitions).Error; err != nil {
		out["status"] = "unavailable"
		return out
	}
	if len(definitions) == 0 {
		return out
	}
	latestAt := int64(0)
	status := SLOStatusAchieved
	for _, definition := range definitions {
		var snapshot database.SLOSnapshot
		if err := h.DB.Where("slo_id = ?", definition.ID).Order("recorded_at DESC, id DESC").First(&snapshot).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				out["status"] = "unavailable"
				return out
			}
			status = SLOStatusInsufficientData
			continue
		}
		if snapshot.RecordedAt > latestAt {
			latestAt = snapshot.RecordedAt
		}
		status = worseSLOStatus(status, snapshot.Status)
	}
	out["status"] = status
	if latestAt > 0 {
		out["lastEvaluatedAt"] = time.Unix(latestAt, 0).UTC().Format(time.RFC3339)
	}
	return out
}

func worseSLOStatus(current, candidate string) string {
	priority := map[string]int{
		SLOStatusAchieved:         1,
		SLOStatusInsufficientData: 2,
		SLOStatusViolated:         3,
	}
	if _, ok := priority[candidate]; !ok {
		candidate = SLOStatusInsufficientData
	}
	if _, ok := priority[current]; !ok {
		current = SLOStatusInsufficientData
	}
	if priority[candidate] > priority[current] {
		return candidate
	}
	return current
}

func overallStatus(obs config.ObservabilityConfig, alerts, evaluation, slo, metrics, telemetry gin.H) string {
	if !obs.Enabled {
		return "disabled"
	}
	if alerts["active"] != int64(0) ||
		alerts["status"] == "unavailable" ||
		evaluation["status"] == alerting.EvaluationFailed ||
		evaluation["status"] == "unavailable" ||
		slo["status"] == SLOStatusViolated ||
		slo["status"] == "unavailable" ||
		metrics["status"] == "unavailable" ||
		metrics["status"] == "unprotected" ||
		telemetry["status"] == "export_degraded" ||
		telemetry["status"] == "incomplete" {
		return "needs_attention"
	}
	if evaluation["status"] == "waiting" ||
		evaluation["status"] == alerting.EvaluationWarmingUp ||
		slo["status"] == "waiting" ||
		slo["status"] == SLOStatusInsufficientData ||
		telemetry["status"] == "export_pending" ||
		telemetry["status"] == "real_backend_deferred" {
		return "waiting"
	}
	return "healthy"
}

func (h *Handler) otlpExporterStatus() string {
	obs := h.obsConfig()
	if !obs.TracingEnabled {
		return "disabled"
	}
	if strings.TrimSpace(obs.OTELExporterOTLPEndpoint) == "" {
		return "real_backend_deferred"
	}
	if !strings.EqualFold(strings.TrimSpace(obs.OTELExporterOTLPProtocol), "http/json") {
		return "incomplete"
	}
	if h == nil || h.Obs == nil || h.Obs.Tracer == nil {
		return "export_degraded"
	}
	if h.Obs.Tracer.ExportBlocked() {
		return "export_degraded"
	}
	if !h.Obs.Tracer.ExportAttempted() {
		return "export_pending"
	}
	return "standard_protocol_ready"
}

// MetricsEndpoint is mounted separately on internal route.
func MetricsEndpoint(obs *observability.Observability) gin.HandlerFunc {
	return func(c *gin.Context) {
		if obs == nil {
			c.String(http.StatusServiceUnavailable, "metrics disabled")
			return
		}
		obs.MetricsHandler().ServeHTTP(c.Writer, c.Request)
	}
}
