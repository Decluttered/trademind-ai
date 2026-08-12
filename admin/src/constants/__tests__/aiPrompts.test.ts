import { describe, expect, it } from 'vitest';
import {
  AI_CONTEXT_PLATFORM_OPTIONS,
  AI_LANGUAGE_OPTIONS,
  AI_TARGET_PLATFORM_OPTIONS,
  AI_TONE_OPTIONS,
} from '../aiPrompts';

describe('AI prompt platform options', () => {
  it('keeps the existing display names as request values', () => {
    expect(AI_TARGET_PLATFORM_OPTIONS.map(({ value }) => value)).toEqual([
      '跨境通用',
      '抖店',
      'TikTok Shop',
      'Shopee',
      'Lazada',
      'Amazon',
    ]);
  });

  it('uses Chinese labels while preserving stable language and tone values', () => {
    expect(AI_LANGUAGE_OPTIONS).toContainEqual({ value: 'en', label: '英语' });
    expect(AI_LANGUAGE_OPTIONS).toContainEqual({ value: 'zh', label: '中文' });
    expect(AI_TONE_OPTIONS).toContainEqual({ value: 'professional', label: '专业稳健' });
    expect(AI_TONE_OPTIONS).toContainEqual({ value: 'friendly', label: '亲切自然' });
  });

  it('keeps internal platform codes for AI conversation context', () => {
    expect(AI_CONTEXT_PLATFORM_OPTIONS).toContainEqual({ value: 'douyin_shop', label: '抖店' });
    expect(AI_CONTEXT_PLATFORM_OPTIONS).toContainEqual({ value: 'tiktok', label: 'TikTok Shop' });
    expect(AI_CONTEXT_PLATFORM_OPTIONS).toContainEqual({ value: 'manual', label: '手动' });
  });
});
