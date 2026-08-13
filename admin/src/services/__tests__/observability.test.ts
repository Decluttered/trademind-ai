import { request } from '@umijs/max';
import { describe, expect, it, vi } from 'vitest';
import {
  acknowledgeAlert,
  fetchObservabilityAlerts,
  silenceAlert,
} from '../observability';

const requestMock = vi.mocked(request);

describe('observability alert API service', () => {
  it('sends stable pagination and alert filters', async () => {
    requestMock.mockResolvedValueOnce({
      code: 0,
      message: 'ok',
      data: {
        items: [],
        pagination: { page: 2, pageSize: 20, total: 0, totalPages: 0 },
      },
    });

    await fetchObservabilityAlerts({
      page: 2,
      pageSize: 20,
      status: 'firing',
      severity: 'critical',
      module: 'worker',
    });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/observability/alerts', {
      method: 'GET',
      params: {
        page: 2,
        pageSize: 20,
        status: 'firing',
        severity: 'critical',
        module: 'worker',
      },
    });
  });

  it('encodes alert ids and sends the required silence audit payload', async () => {
    requestMock.mockResolvedValue({ code: 0, message: 'ok', data: {} });

    await acknowledgeAlert('alert/1');
    await silenceAlert('alert/1', { reason: '计划维护窗口', durationHours: 8 });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/observability/alerts/alert%2F1/ack', {
      method: 'POST',
      data: {},
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/observability/alerts/alert%2F1/silence', {
      method: 'POST',
      data: { reason: '计划维护窗口', durationHours: 8 },
    });
  });
});
