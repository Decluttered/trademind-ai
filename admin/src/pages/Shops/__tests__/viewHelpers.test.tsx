import { describe, expect, it } from 'vitest';

import { buildShopAuthPayload, formatPlatformPartnerErr, isStandardAuthField } from '../viewHelpers';

describe('shop view helpers', () => {
  it('keeps standard auth fields at the envelope root', () => {
    expect(
      buildShopAuthPayload(
        {
          authType: 'oauth',
          appKey: 'key',
          accessToken: 'token',
          expiresAt: '2026-08-11T00:00:00Z',
          customRegion: 'eu',
          emptyValue: '',
        },
        'token',
      ),
    ).toEqual({
      authType: 'oauth',
      appKey: 'key',
      accessToken: 'token',
      expiresAt: '2026-08-11T00:00:00Z',
      authConfig: { customRegion: 'eu' },
    });
  });

  it('falls back to the provider auth type and identifies custom fields', () => {
    expect(buildShopAuthPayload({ customField: 'value' }, 'api_key')).toEqual({
      authType: 'api_key',
      authConfig: { customField: 'value' },
    });
    expect(isStandardAuthField('appSecret')).toBe(true);
    expect(isStandardAuthField('customField')).toBe(false);
  });

  it('adds actionable context for known configuration errors', () => {
    expect(formatPlatformPartnerErr(new Error('platform_tiktok config incomplete'))).toContain('平台接入设置');
    expect(formatPlatformPartnerErr(new Error('unexpected provider failure'))).toBe('unexpected provider failure');
  });
});
