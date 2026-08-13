import { request } from '@umijs/max';
import { getWithParams, postJSON } from '@/services/request';

export type ObservabilityOverview = {
  enabled: boolean;
  mode: string;
  metricsEnabled: boolean;
  tracingEnabled: boolean;
  alertingEnabled: boolean;
  metricsPath: string;
  metricsInternal: boolean;
  otelExportBlocked: boolean;
  runtimeStatus?: {
    otlpExporter?: string;
    otlpProtocol?: string;
    mockCollectorVerification?: string;
    [key: string]: string | number | undefined;
  };
  telemetry?: {
    dropped?: number;
    exportFailures?: number;
    exportSuccess?: number;
  };
  environment: string;
  timestamp: string;
};

export type AlertEvent = {
  id: string;
  ruleId: string;
  severity: string;
  status: string;
  module: string;
  summary: string;
  occurrenceCount: number;
  lastSeenAt: string;
};

export type AlertListResult = {
  items: AlertEvent[];
  pagination: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
};

export async function fetchObservabilityOverview() {
  return request<{ data: ObservabilityOverview }>('/api/v1/observability/overview', { method: 'GET' });
}

export async function fetchObservabilityAlerts(params?: {
  status?: string;
  severity?: string;
  module?: string;
  page?: number;
  pageSize?: number;
  limit?: number;
}) {
  return getWithParams<AlertListResult>('/api/v1/observability/alerts', params);
}

export async function acknowledgeAlert(id: string) {
  return postJSON<{ id: string; status: string }>(
    `/api/v1/observability/alerts/${encodeURIComponent(id)}/ack`,
    {},
  );
}

export async function silenceAlert(id: string, body: { reason?: string; durationHours?: number }) {
  return postJSON<{ id: string; status: string; expiresAt: string }>(
    `/api/v1/observability/alerts/${encodeURIComponent(id)}/silence`,
    body,
  );
}
