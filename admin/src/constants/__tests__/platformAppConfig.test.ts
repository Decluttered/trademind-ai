import { describe, expect, it } from 'vitest';
import { platformAppFieldHelp, platformAppFieldLabel, platformAppFieldPlaceholder } from '../platformAppConfig';

describe('platform app field copy', () => {
  it('keeps generic redirect_uri as a callback URL for other platforms', () => {
    expect(platformAppFieldLabel({ name: 'redirect_uri', label: 'Redirect URI', type: 'text', required: true, sensitive: false })).toBe(
      '授权回调地址',
    );
  });

  it('labels eBay redirect_uri as RuName instead of an https callback', () => {
    const field = { name: 'redirect_uri', label: 'Redirect URI', type: 'text', required: true, sensitive: false };
    expect(platformAppFieldLabel(field, 'ebay')).toContain('RuName');
    expect(platformAppFieldHelp(field, 'ebay')).toContain('RuName');
    expect(platformAppFieldPlaceholder(field, 'ebay')).toContain('SB');
  });
});
