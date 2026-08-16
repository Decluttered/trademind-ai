# P5 Observability Architecture

```text
HTTP / Webhook / Worker
  → Context Correlation (request_id, trace_id, tenant_id, task_id)
  → Structured Logger (JSON production)
  → Metrics Registry (Prometheus, low-cardinality labels)
  → Tracer (OpenTelemetry, safe attributes)
  → Exporter /internal/metrics (internal only)
  → Dashboard + Alert Rules + Admin Observability Center
```

Unified facade: `backend/internal/pkg/observability`. A second logging/metrics system is prohibited.
