import { describe, expect, it } from 'vitest';
import {
  resolveAlertSource,
  systemAlertSeverityMeta,
  systemAlertStatusMeta,
} from './alertMeta';
import { createActionGuard } from './actionGuard';

describe('alert center helpers', () => {
  it('restores a permitted URL source and falls back to the first accessible source', () => {
    expect(resolveAlertSource('system', { business: true, system: true })).toBe('system');
    expect(resolveAlertSource('unknown', { business: true, system: true })).toBe('business');
    expect(resolveAlertSource('business', { business: false, system: true })).toBe('system');
  });

  it('maps system severity and status to user-facing labels', () => {
    expect(systemAlertSeverityMeta('critical')).toEqual({ color: 'red', text: '严重' });
    expect(systemAlertStatusMeta('acknowledged')).toEqual({ color: 'blue', text: '已确认' });
    expect(systemAlertStatusMeta('custom').text).toBe('custom');
  });

  it('rejects duplicate action locks until the active action completes', () => {
    const guard = createActionGuard();
    expect(guard.tryLock('alert-1')).toBe(true);
    expect(guard.tryLock('alert-1')).toBe(false);
    expect(guard.isLocked('alert-1')).toBe(true);
    guard.unlock('alert-1');
    expect(guard.tryLock('alert-1')).toBe(true);
  });
});
