import { describe, expect, it } from 'vitest';
import {
  NOTIFICATION_CHANNEL_OPTIONS,
} from '@/constants/alertNotify';

describe('notification channel options', () => {
  it('exposes every implemented outbound channel', () => {
    expect(NOTIFICATION_CHANNEL_OPTIONS.map((option) => option.value)).toEqual([
      'mail',
      'webhook',
      'feishu',
      'wecom',
    ]);
  });
});
