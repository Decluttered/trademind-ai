import { request } from '@umijs/max';
import { getWithParams, postJSON } from '@/services/request';

export type ObservabilityOverview = {
  overallStatus: 'healthy' | 'needs_attention' | 'waiting' | 'disabled';
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
    [key: string]: string | number | undefined;
  };
  metrics?: {
    status: 'active' | 'disabled' | 'unavailable' | 'unprotected';
    path: string;
    internalOnly: boolean;
    allowlistConfigured: boolean;
  };
  alerts?: {
    status: 'active' | 'disabled' | 'unavailable';
    active: number;
    critical: number;
    warning: number;
  };
  evaluation?: {
    status: 'succeeded' | 'failed' | 'warming_up' | 'waiting' | 'disabled' | 'unavailable';
    lastEvaluatedAt?: string;
    rulesChecked: number;
    rulesSkipped: number;
    alertsFired: number;
    alertsResolved: number;
  };
  slo?: {
    status: 'achieved' | 'violated' | 'insufficient_data' | 'waiting' | 'disabled' | 'unavailable';
    lastEvaluatedAt?: string;
  };
  telemetry?: {
    status?: string;
    protocol?: string;
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
