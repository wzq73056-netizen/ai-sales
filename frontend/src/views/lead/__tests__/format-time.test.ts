import { describe, expect, it } from 'vitest';

/** 与线索池页面一致的 ISO 时间格式化 */
export function formatTime(value: string) {
  if (!value) return '-';

  return String(value)
    .replace('T', ' ')
    .replace(/\.\d+Z?$/, '')
    .replace(/[+-]\d{2}:\d{2}$/, '');
}

describe('formatTime', () => {
  it('空值返回占位符', () => {
    expect(formatTime('')).toBe('-');
  });

  it('去掉 ISO 的 T 和毫秒', () => {
    expect(formatTime('2026-11-06T17:20:31.123Z')).toBe('2026-11-06 17:20:31');
  });

  it('去掉时区后缀', () => {
    expect(formatTime('2026-11-06T17:20:31+08:00')).toBe('2026-11-06 17:20:31');
  });

  it('普通时间原样返回', () => {
    expect(formatTime('2026-11-06 17:20:31')).toBe('2026-11-06 17:20:31');
  });
});
