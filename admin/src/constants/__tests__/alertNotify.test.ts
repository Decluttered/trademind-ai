import { describe, expect, it } from 'vitest';
import {
  NOTIFICATION_CHANNEL_OPTIONS,
  isNotificationChannelAvailable,
} from '@/constants/alertNotify';

describe('notification channel production visibility', () => {
  it('keeps supported channels and excludes planned channels in production', () => {
    const visible = NOTIFICATION_CHANNEL_OPTIONS.filter((option) =>
      isNotificationChannelAvailable(option.value, true),
    ).map((option) => option.value);

    expect(visible).toEqual(['mail', 'webhook']);
  });

  it('keeps planned channels available for development verification', () => {
    expect(isNotificationChannelAvailable('feishu', false)).toBe(true);
    expect(isNotificationChannelAvailable('wecom', false)).toBe(true);
  });

  it('fails closed for unknown production channels', () => {
    expect(isNotificationChannelAvailable('unknown', true)).toBe(false);
  });
});
