package alerting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/trademind-ai/trademind/backend/internal/modules/operationlog"
	"gorm.io/gorm"
)

func TestAlertListItemJSONContract(t *testing.T) {
	lastSeenAt := time.Date(2026, time.August, 10, 12, 30, 0, 0, time.UTC)
	payload, err := json.Marshal(newAlertListItem(AlertEvent{
		ID:              "alert-e2e-1",
		RuleID:          "http_5xx_elevated",
		Severity:        SeverityWarning,
		Status:          StatusFiring,
		Module:          "http",
		Summary:         "5xx spike",
		OccurrenceCount: 2,
		LastSeenAt:      lastSeenAt,
		Fingerprint:     "must-not-be-exposed",
		SafeDetails:     "must-not-be-exposed",
		TenantScope:     "must-not-be-exposed",
	}))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"id", "ruleId", "severity", "status", "module", "summary", "occurrenceCount", "lastSeenAt"}
	if len(got) != len(wantKeys) {
		t.Fatalf("unexpected alert list contract: %s", payload)
	}
	for _, key := range wantKeys {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing lowerCamelCase key %q in %s", key, payload)
		}
	}
	if got["id"] != "alert-e2e-1" || got["ruleId"] != "http_5xx_elevated" {
		t.Fatalf("unexpected alert identity fields: %s", payload)
	}
}

func newAlertHandlerTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&AlertEvent{}, &AlertSilence{}, &operationlog.OperationLog{}))
	router := gin.New()
	Register(router.Group("/api/v1"), &Handler{
		Svc:   NewService(db, time.Minute, true),
		OpLog: &operationlog.Service{DB: db},
	})
	return router, db
}

func TestAlertListSupportsStablePaginationAndFilters(t *testing.T) {
	router, db := newAlertHandlerTestRouter(t)
	rows := []AlertEvent{
		{ID: "alert-1", RuleID: "rule-1", Severity: SeverityWarning, Status: StatusFiring, Module: "http", Summary: "older", LastSeenAt: time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)},
		{ID: "alert-2", RuleID: "rule-2", Severity: SeverityCritical, Status: StatusFiring, Module: "worker", Summary: "newer", LastSeenAt: time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)},
		{ID: "alert-3", RuleID: "rule-3", Severity: SeverityCritical, Status: StatusResolved, Module: "worker", Summary: "resolved", LastSeenAt: time.Date(2026, 8, 10, 11, 0, 0, 0, time.UTC)},
	}
	require.NoError(t, db.Create(&rows).Error)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/observability/alerts?status=firing&severity=critical&module=worker&page=1&pageSize=1", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"items":[{"id":"alert-2"`)
	require.Contains(t, recorder.Body.String(), `"pagination":{"page":1,"pageSize":1,"total":1,"totalPages":1}`)
}

func TestAlertMutationsValidateInputAndMissingRows(t *testing.T) {
	router, _ := newAlertHandlerTestRouter(t)

	invalidSilence := httptest.NewRecorder()
	router.ServeHTTP(invalidSilence, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/observability/alerts/missing/silence",
		strings.NewReader(`{"reason":"","durationHours":0}`),
	))
	require.Equal(t, http.StatusBadRequest, invalidSilence.Code)

	missingAck := httptest.NewRecorder()
	router.ServeHTTP(missingAck, httptest.NewRequest(http.MethodPost, "/api/v1/observability/alerts/missing/ack", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusNotFound, missingAck.Code)
}

func TestAlertSilenceUpdatesStatusAndCreatesAuditRecord(t *testing.T) {
	router, db := newAlertHandlerTestRouter(t)
	row := AlertEvent{
		ID:         "alert-silence",
		RuleID:     "worker_backlog",
		Severity:   SeverityCritical,
		Status:     StatusFiring,
		Module:     "worker",
		Summary:    "backlog",
		LastSeenAt: time.Now().UTC(),
	}
	require.NoError(t, db.Create(&row).Error)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/observability/alerts/alert-silence/silence",
		strings.NewReader(`{"reason":"maintenance window","durationHours":8}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var updated AlertEvent
	require.NoError(t, db.First(&updated, "id = ?", row.ID).Error)
	require.Equal(t, StatusSilenced, updated.Status)
	var silence AlertSilence
	require.NoError(t, db.First(&silence, "alert_id = ?", row.ID).Error)
	require.Equal(t, "maintenance window", silence.Reason)
	require.Equal(t, row.RuleID, silence.RuleID)
	var audit operationlog.OperationLog
	require.NoError(t, db.First(&audit, "resource_id = ?", row.ID).Error)
	require.Equal(t, "alert.silence", audit.Action)
	require.Equal(t, "alert_event", audit.Resource)
	require.Equal(t, "alerts.silence", audit.Permission)
}

func TestAlertAcknowledgeCreatesAuditRecord(t *testing.T) {
	router, db := newAlertHandlerTestRouter(t)
	row := AlertEvent{
		ID:         "alert-ack",
		RuleID:     "http_5xx_elevated",
		Severity:   SeverityWarning,
		Status:     StatusFiring,
		Module:     "http",
		Summary:    "5xx spike",
		LastSeenAt: time.Now().UTC(),
	}
	require.NoError(t, db.Create(&row).Error)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/observability/alerts/alert-ack/ack",
		strings.NewReader(`{}`),
	))
	require.Equal(t, http.StatusOK, recorder.Code)

	var audit operationlog.OperationLog
	require.NoError(t, db.First(&audit, "resource_id = ?", row.ID).Error)
	require.Equal(t, "alert.acknowledge", audit.Action)
	require.Equal(t, "alert_event", audit.Resource)
	require.Equal(t, "alerts.ack", audit.Permission)
}
