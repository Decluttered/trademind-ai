import { request } from '@umijs/max';
import { describe, expect, it, vi } from 'vitest';
import {
  getCustomerAutoReplyPolicy,
  getCustomerAutoReplySetting,
  queryCustomerAutoReplyRuns,
  sendPlatformMessage,
  updateCustomerAutoReplyPolicy,
  updateCustomerAutoReplySetting,
} from '../customer';

const requestMock = vi.mocked(request);

describe('customer service API contracts', () => {
  it('sends the required client message id for platform delivery', async () => {
    requestMock.mockResolvedValueOnce({ code: 0, message: 'ok', data: { id: 'message-1' } });

    await sendPlatformMessage('conversation-1', {
      reply: 'Hello',
      clientMessageId: 'client-1',
      suggestionId: 'suggestion-1',
    });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/conversations/conversation-1/send-platform-message', {
      method: 'POST',
      data: { reply: 'Hello', clientMessageId: 'client-1', suggestionId: 'suggestion-1' },
    });
  });

  it('uses shop-scoped policy and run endpoints', async () => {
    requestMock.mockResolvedValue({ code: 0, message: 'ok', data: [] });

    await getCustomerAutoReplyPolicy('shop-1');
    await queryCustomerAutoReplyRuns('shop-1');
    await updateCustomerAutoReplyPolicy('shop-1', {
      enabled: true,
      tone: 'professional',
      shopPolicy: 'Confirmed policy',
      maxReplyRunes: 600,
      maxRepliesPerHour: 20,
      requireOrderContext: true,
      lowRiskOnly: true,
    });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/shops/shop-1/auto-reply-policy', { method: 'GET' });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/shops/shop-1/auto-reply-runs', { method: 'GET' });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/shops/shop-1/auto-reply-policy', {
      method: 'PUT',
      data: expect.objectContaining({ enabled: true, lowRiskOnly: true, requireOrderContext: true }),
    });
  });

  it('uses the tenant runtime settings endpoint', async () => {
    requestMock.mockResolvedValue({ code: 0, message: 'ok', data: {} });

    await getCustomerAutoReplySetting();
    await updateCustomerAutoReplySetting({
      messageSyncEnabled: true,
      autoReplyEnabled: false,
      pollIntervalSeconds: 60,
    });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/auto-reply-setting', { method: 'GET' });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/customer/auto-reply-setting', {
      method: 'PUT',
      data: { messageSyncEnabled: true, autoReplyEnabled: false, pollIntervalSeconds: 60 },
    });
  });
});
