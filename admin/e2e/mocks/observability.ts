import { ok } from './envelope';

export function observabilityResponse(path: string) {
  if (path === '/api/v1/observability/overview') {
    return ok({
      overallStatus: 'needs_attention',
      enabled: true,
      mode: 'hybrid',
      metricsEnabled: true,
      tracingEnabled: true,
      alertingEnabled: true,
      metricsPath: '/internal/metrics',
      metricsInternal: true,
      otelExportBlocked: false,
      runtimeStatus: {
        otlpExporter: 'real_backend_deferred',
        otlpProtocol: 'http/json',
      },
      metrics: {
        status: 'active',
        path: '/internal/metrics',
        internalOnly: true,
        allowlistConfigured: true,
      },
      alerts: { status: 'active', active: 2, critical: 1, warning: 1 },
      evaluation: {
        status: 'succeeded',
        lastEvaluatedAt: '2026-08-10T12:29:00Z',
        rulesChecked: 16,
        rulesSkipped: 0,
        alertsFired: 1,
        alertsResolved: 0,
      },
      slo: { status: 'achieved', lastEvaluatedAt: '2026-08-10T12:29:00Z' },
      telemetry: {
        status: 'real_backend_deferred',
        protocol: 'http/json',
        dropped: 0,
        exportFailures: 0,
        exportSuccess: 0,
      },
      environment: 'e2e',
      timestamp: '2026-08-10T12:30:00Z',
    });
  }
  if (path === '/api/v1/observability/alerts') {
    return ok({
      items: [
        {
          id: 'alert-e2e-1',
          ruleId: 'http_5xx_elevated',
          severity: 'warning',
          status: 'firing',
          module: 'http',
          summary: 'E2E API error rate alert',
          occurrenceCount: 2,
          lastSeenAt: '2026-08-10T12:30:00Z',
        },
        {
          id: 'alert-e2e-2',
          ruleId: 'worker_backlog',
          severity: 'critical',
          status: 'acknowledged',
          module: 'worker',
          summary: 'E2E worker backlog alert',
          occurrenceCount: 1,
          lastSeenAt: '2026-08-10T12:25:00Z',
        },
      ],
      pagination: { page: 1, pageSize: 20, total: 2, totalPages: 1 },
    });
  }
  return null;
}
