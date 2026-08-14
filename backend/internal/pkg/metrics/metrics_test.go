package metrics

import (
	"math"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestForbiddenLabelKeys(t *testing.T) {
	v := NewLabelValidator(10)
	for _, k := range []string{"request_id", "user_id", "task_id", "tenant_id"} {
		if err := v.ValidateKey(k); err == nil {
			t.Fatalf("expected forbidden label %s", k)
		}
	}
}

func TestSnapshotPreservesLabelsAndCalculatesWindowValues(t *testing.T) {
	reg := NewRegistry("")
	counter, err := reg.Counter("requests_total", "requests", "result")
	if err != nil {
		t.Fatal(err)
	}
	histogram, err := reg.Histogram("latency_seconds", "latency", []float64{0.1, 0.5, 1}, "route")
	if err != nil {
		t.Fatal(err)
	}
	counter.WithLabelValues("success").Add(10)
	counter.WithLabelValues("failure").Add(2)
	histogram.WithLabelValues("api").Observe(0.1)
	before := reg.Snapshot(time.Unix(100, 0))

	counter.WithLabelValues("success").Add(8)
	counter.WithLabelValues("failure").Add(2)
	for _, value := range []float64{0.1, 0.2, 0.4, 0.7} {
		histogram.WithLabelValues("api").Observe(value)
	}
	after := reg.Snapshot(time.Unix(400, 0))

	failures, ok := CounterDelta(before, after, "requests_total", map[string]string{"result": "failure"})
	if !ok || failures != 2 {
		t.Fatalf("failure delta = %v, ok=%v", failures, ok)
	}
	total, ok := CounterDelta(before, after, "requests_total", nil)
	if !ok || total != 10 {
		t.Fatalf("total delta = %v, ok=%v", total, ok)
	}
	p95, count, ok := HistogramQuantileDelta(before, after, "latency_seconds", map[string]string{"route": "api"}, 0.95)
	if !ok || count != 4 || p95 != 1 {
		t.Fatalf("p95 = %v, count=%d, ok=%v", p95, count, ok)
	}
	within, totalCount, ok := HistogramThresholdDelta(before, after, "latency_seconds", map[string]string{"route": "api"}, 0.5)
	if !ok || within != 3 || totalCount != 4 {
		t.Fatalf("threshold count = %d/%d, ok=%v", within, totalCount, ok)
	}
}

func TestCounterDeltaFailsClosedOnReset(t *testing.T) {
	previous := Snapshot{Families: map[string]FamilySnapshot{
		"jobs_total": {Type: dto.MetricType_COUNTER, Samples: []Sample{{Labels: map[string]string{"result": "failure"}, Value: 9}}},
	}}
	current := Snapshot{Families: map[string]FamilySnapshot{
		"jobs_total": {Type: dto.MetricType_COUNTER, Samples: []Sample{{Labels: map[string]string{"result": "failure"}, Value: 1}}},
	}}
	if value, ok := CounterDelta(previous, current, "jobs_total", nil); ok || math.Abs(value) > 0 {
		t.Fatalf("reset must be insufficient, value=%v ok=%v", value, ok)
	}
}

func TestCardinalityDoesNotGrowWithIDs(t *testing.T) {
	v := NewLabelValidator(256)
	reg := NewRegistry("test")
	cat, err := RegisterCatalog(reg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		_ = v.Validate("method", "GET") // allowed key
		if err := v.Validate("request_id", "x"); err == nil {
			t.Fatal("request_id should be forbidden")
		}
	}
	if cat.HTTPRequestsTotal != nil {
		cat.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/health", "2xx", "success").Inc()
	}
	if reg.Validator().SeriesCount() > 300 {
		t.Fatalf("too many series: %d", reg.Validator().SeriesCount())
	}
}

func TestNormalizeResult(t *testing.T) {
	if NormalizeResult("provider_timeout") != "timeout" {
		t.Fatal("expected timeout")
	}
	if NormalizeResult("environment_blocked") != "environment_blocked" {
		t.Fatal("expected environment_blocked")
	}
}
