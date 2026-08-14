import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AppMessageBridge from '@/components/AppMessageBridge';
import ObservabilityCenterPage from './index';

const observabilityService = vi.hoisted(() => ({
  fetchObservabilityOverview: vi.fn(),
}));

vi.mock('@/services/observability', () => observabilityService);

const overview = {
  overallStatus: 'needs_attention' as const,
  enabled: true,
  mode: 'hybrid',
  metricsEnabled: true,
  tracingEnabled: true,
  alertingEnabled: true,
  metricsPath: '/internal/metrics',
  metricsInternal: true,
  otelExportBlocked: false,
  metrics: {
    status: 'active' as const,
    path: '/internal/metrics',
    internalOnly: true,
    allowlistConfigured: true,
  },
  alerts: { status: 'active' as const, active: 3, critical: 1, warning: 2 },
  evaluation: {
    status: 'succeeded' as const,
    lastEvaluatedAt: '2026-08-14T10:00:00Z',
    rulesChecked: 16,
    rulesSkipped: 0,
    alertsFired: 1,
    alertsResolved: 2,
  },
  slo: { status: 'achieved' as const, lastEvaluatedAt: '2026-08-14T10:00:00Z' },
  telemetry: {
    status: 'real_backend_deferred',
    protocol: 'http/json',
    exportSuccess: 0,
    exportFailures: 0,
    dropped: 0,
  },
  environment: 'production',
  timestamp: '2026-08-14T10:01:00Z',
};

beforeEach(() => {
  vi.clearAllMocks();
	Object.defineProperty(window, 'matchMedia', {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
  observabilityService.fetchObservabilityOverview.mockResolvedValue({ data: overview });
});

describe('ObservabilityCenterPage', () => {
  it('renders operational state from the overview response', async () => {
    renderObservabilityPage();

    expect(await screen.findByText('需要处理')).toBeInTheDocument();
    expect(screen.getByText('活跃系统告警')).toBeInTheDocument();
    expect(screen.getByText('严重 1 · 警告 2')).toBeInTheDocument();
    expect(screen.getAllByText('最近评估成功').length).toBeGreaterThan(0);
    expect(screen.getAllByText('未配置导出后端').length).toBeGreaterThan(0);
    expect(screen.getByText('/internal/metrics')).toBeInTheDocument();
  });

  it('keeps the last successful data when a refresh fails', async () => {
    observabilityService.fetchObservabilityOverview
      .mockResolvedValueOnce({ data: overview })
      .mockRejectedValueOnce(new Error('offline'));
    renderObservabilityPage();

    expect(await screen.findByText('需要处理')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '刷新运行状态' }));

    expect(await screen.findByText('可观测性概览加载失败')).toBeInTheDocument();
    expect(screen.getByText('活跃系统告警')).toBeInTheDocument();
    await waitFor(() => expect(observabilityService.fetchObservabilityOverview).toHaveBeenCalledTimes(2));
  });

  it('does not render placeholder operational values when the initial request fails', async () => {
    observabilityService.fetchObservabilityOverview.mockRejectedValueOnce(new Error('offline'));
    renderObservabilityPage();

    expect(await screen.findByText('可观测性概览加载失败')).toBeInTheDocument();
    expect(screen.queryByText('活跃系统告警')).not.toBeInTheDocument();
    expect(screen.queryByText('环境：未知')).not.toBeInTheDocument();
  });
});

function renderObservabilityPage() {
  return render(
    <App>
      <AppMessageBridge />
      <ObservabilityCenterPage />
    </App>,
  );
}
