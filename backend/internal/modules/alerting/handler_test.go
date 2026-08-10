package alerting

import (
	"encoding/json"
	"testing"
	"time"
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
