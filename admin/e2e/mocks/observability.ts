import { ok } from './envelope';

export function observabilityResponse(path: string) {
  if (path === '/api/v1/observability/overview') {
    return ok({
      enabled: true,
      mode: 'local',
      metricsEnabled: true,
      tracingEnabled: false,
      alertingEnabled: true,
      metricsPath: '/internal/metrics',
      metricsInternal: true,
      otelExportBlocked: true,
      runtimeStatus: {
        otlpExporter: 'disabled',
        otlpProtocol: 'http/json',
      },
      telemetry: { dropped: 0, exportFailures: 0, exportSuccess: 0 },
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
    });
  }
  return null;
}
